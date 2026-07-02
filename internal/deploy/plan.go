package deploy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Plan(opts Options) (Result, error) {
	opts = withDefaults(opts)
	if err := validate(opts); err != nil {
		return Result{}, err
	}

	contextDir, err := filepath.Abs(opts.CWD)
	if err != nil {
		return Result{}, fmt.Errorf("resolve cwd: %w", err)
	}
	dockerfile := filepath.Join(contextDir, "Dockerfile")
	info, err := os.Stat(dockerfile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{}, fmt.Errorf("Dockerfile not found in cwd: %s", dockerfile)
		}
		return Result{}, fmt.Errorf("stat Dockerfile: %w", err)
	}
	if info.IsDir() {
		return Result{}, fmt.Errorf("Dockerfile path is a directory: %s", dockerfile)
	}
	if opts.Port == 0 {
		opts.Port = exposedPort(dockerfile)
	}
	envFilePath := ""
	if opts.EnvFile != "" {
		envFilePath, err = filepath.Abs(opts.EnvFile)
		if err != nil {
			return Result{}, fmt.Errorf("resolve env file: %w", err)
		}
		if info, err := os.Stat(envFilePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return Result{}, fmt.Errorf("env file not found: %s", envFilePath)
			}
			return Result{}, fmt.Errorf("stat env file: %w", err)
		} else if info.IsDir() {
			return Result{}, fmt.Errorf("env file path is a directory: %s", envFilePath)
		}
	}

	host := opts.ServiceName + "." + opts.Domain
	image := imageName(opts)
	manifest := manifestFor(opts, host, image)
	commands := []string{
		fmt.Sprintf("docker build -f %s -t %s %s", dockerfile, image, contextDir),
		loadOrPushCommand(opts, image),
	}
	if envFilePath != "" {
		commands = append(commands, fmt.Sprintf("kubectl create secret generic %s-env -n %s --from-env-file=%s --dry-run=client -o yaml | kubectl apply -f -", opts.ServiceName, opts.Namespace, envFilePath))
	}
	commands = append(commands,
		"kubectl apply -f <generated manifest>",
		fmt.Sprintf("kubectl rollout status deployment/%s -n %s --timeout=180s", opts.ServiceName, opts.Namespace),
	)
	if opts.Exposure != "internet" && shouldSyncServiceDNS(opts) {
		commands = append(commands, serviceDNSCommand(opts, host))
	}
	if opts.Exposure == "internet" {
		commands = append(commands, cloudflareTunnelExposeCommand(opts, host))
	}

	return Result{
		ServiceName:    opts.ServiceName,
		Host:           host,
		Image:          image,
		Namespace:      opts.Namespace,
		DockerfilePath: dockerfile,
		ContextDir:     contextDir,
		EnvFilePath:    envFilePath,
		Port:           opts.Port,
		Exposure:       opts.Exposure,
		TailscaleOnly:  opts.Exposure == "tailscale",
		DryRun:         opts.DryRun,
		Commands:       commands,
		Manifest:       manifest,
	}, nil
}

func withDefaults(opts Options) Options {
	if opts.CWD == "" {
		opts.CWD = "."
	}
	if opts.Namespace == "" {
		opts.Namespace = "ship-services"
	}
	if opts.Domain == "" {
		opts.Domain = "example.com"
	}
	if opts.GatewayNamespace == "" {
		opts.GatewayNamespace = "ship-system"
	}
	if opts.GatewayName == "" {
		opts.GatewayName = "ship-tailscale"
	}
	if opts.InternetGateway == "" {
		opts.InternetGateway = "ship-internet"
	}
	if opts.Exposure == "" {
		opts.Exposure = "tailscale"
	}
	if opts.ImagePrefix == "" {
		opts.ImagePrefix = "ship"
	}
	if opts.ImageTag == "" {
		opts.ImageTag = time.Now().UTC().Format("20060102150405")
	}
	if opts.KindCluster == "" {
		opts.KindCluster = "ship"
	}
	if opts.DNSMode == "" {
		opts.DNSMode = "manual"
	}
	return opts
}

func exposedPort(dockerfile string) int {
	raw, err := os.ReadFile(dockerfile)
	if err != nil {
		return 8080
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.EqualFold(fields[0], "EXPOSE") {
			continue
		}
		port := strings.TrimSuffix(fields[1], "/tcp")
		var value int
		if _, err := fmt.Sscanf(port, "%d", &value); err == nil && value > 0 && value <= 65535 {
			return value
		}
	}
	return 8080
}

func validate(opts Options) error {
	if opts.ServiceName == "" {
		return errors.New("--service is required")
	}
	if !serviceNamePattern.MatchString(opts.ServiceName) {
		return errors.New("service name must be DNS-safe: lowercase letters, numbers, hyphens")
	}
	switch opts.ServiceName {
	case "www", "api", "admin":
		return fmt.Errorf("reserved service name: %s", opts.ServiceName)
	}
	if opts.Port < 0 || opts.Port > 65535 {
		return errors.New("port must be between 0 and 65535")
	}
	switch opts.Exposure {
	case "tailscale", "internet":
	default:
		return errors.New("exposure must be tailscale or internet")
	}
	switch opts.DNSMode {
	case "manual", "auto", "cloudflare":
	default:
		return errors.New("dns mode must be manual, auto, or cloudflare")
	}
	if opts.DNSMode == "cloudflare" && !opts.DryRun && !opts.CloudflareToken {
		return errors.New("missing CLOUDFLARE_API_TOKEN with Zone DNS Edit")
	}
	if opts.Exposure == "internet" && !opts.DryRun {
		if !opts.CloudflareToken {
			return errors.New("missing CLOUDFLARE_API_TOKEN with Zone DNS Edit and Cloudflare Tunnel Edit")
		}
		if opts.CloudflareAccountID == "" {
			return errors.New("missing CLOUDFLARE_ACCOUNT_ID for internet exposure")
		}
		if opts.CloudflareTunnelID == "" {
			return errors.New("missing CLOUDFLARE_TUNNEL_ID; run ship install")
		}
	}
	return nil
}

func imageName(opts Options) string {
	base := strings.TrimRight(opts.ImagePrefix, "/") + "/" + opts.ServiceName
	if opts.Registry != "" {
		return strings.TrimRight(opts.Registry, "/") + "/" + base + ":" + opts.ImageTag
	}
	return base + ":" + opts.ImageTag
}

func loadOrPushCommand(opts Options, image string) string {
	if opts.Registry != "" {
		return "docker push " + image
	}
	return fmt.Sprintf("kind load docker-image --name %s %s", opts.KindCluster, image)
}

func shouldSyncServiceDNS(opts Options) bool {
	return opts.DNSMode == "cloudflare" || (opts.DNSMode == "auto" && opts.CloudflareToken)
}

func serviceDNSCommand(opts Options, host string) string {
	ship := opts.ShipCommand
	if ship == "" {
		ship = "ship"
	}
	return fmt.Sprintf(
		"%s dns publish --record %s --target $(kubectl get gateway %s -n %s -o jsonpath=%s)",
		shellQuote(ship),
		shellQuote(host),
		shellQuote(opts.GatewayName),
		shellQuote(opts.GatewayNamespace),
		shellQuote("{.status.addresses[0].value}"),
	)
}

func cloudflareTunnelExposeCommand(opts Options, host string) string {
	serviceURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:80", opts.ServiceName, opts.Namespace)
	return fmt.Sprintf("cloudflare tunnel expose %s -> %s using tunnel %s", host, serviceURL, opts.CloudflareTunnelID)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
