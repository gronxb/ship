package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var composeFileNames = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

type composeModel struct {
	Services map[string]composeService `json:"services"`
}

type composeService struct {
	Ports []composePort `json:"ports"`
}

type composePort struct {
	Target    int             `json:"target"`
	Published json.RawMessage `json:"published"`
	Protocol  string          `json:"protocol"`
	HostIP    string          `json:"host_ip"`
}

func planCompose(ctx context.Context, opts Options, contextDir string) (Result, error) {
	composePath, err := resolveComposeFile(contextDir, opts.ComposeFile)
	if err != nil {
		return Result{}, err
	}
	envPath, err := resolveOptionalFile(opts.EnvFile)
	if err != nil {
		return Result{}, err
	}
	args := composeArgs(composePath, contextDir, envPath)
	if err := validateComposeConfig(ctx, args); err != nil {
		return Result{}, err
	}
	config, err := readComposeConfig(ctx, args)
	if err != nil {
		return Result{}, err
	}
	serviceName, service, err := selectComposeService(config.Services, opts.ComposeService)
	if err != nil {
		return Result{}, err
	}
	publishedPort, err := selectComposePort(serviceName, service.Ports, opts.Port)
	if err != nil {
		return Result{}, err
	}
	hostGateway, err := resolveKindHostGatewayContext(ctx, opts.KindCluster)
	if err != nil {
		return Result{}, err
	}
	host := opts.ServiceName + "." + opts.Domain
	manifest := composeManifestFor(opts, host, hostGateway, publishedPort)
	prefix := composeCommand(args)
	commands := []string{
		prefix + " config --quiet",
		prefix + " up -d --wait --wait-timeout 180",
		prefix + " ps --all --format json",
		"kubectl apply -f <generated manifest>",
	}
	if opts.Exposure != "internet" && shouldSyncServiceDNS(opts) {
		commands = append(commands, serviceDNSCommand(opts, host))
	}
	if opts.Exposure == "internet" {
		commands = append(commands, cloudflareTunnelExposeCommand(opts, host))
	}
	return Result{
		ServiceName:     opts.ServiceName,
		Runtime:         "compose",
		Host:            host,
		Namespace:       opts.Namespace,
		ComposeFilePath: composePath,
		ComposeService:  serviceName,
		PublishedPort:   publishedPort,
		HostGateway:     hostGateway,
		ContextDir:      contextDir,
		EnvFilePath:     envPath,
		Port:            publishedPort,
		Exposure:        opts.Exposure,
		TailscaleOnly:   opts.Exposure == "tailscale",
		DryRun:          opts.DryRun,
		Commands:        commands,
		Manifest:        manifest,
	}, nil
}

func resolveComposeFile(contextDir string, requested string) (string, error) {
	if requested != "" {
		path := requested
		if !filepath.IsAbs(path) {
			path = filepath.Join(contextDir, path)
		}
		return requireRegularFile(path, "Compose file")
	}
	for _, name := range composeFileNames {
		path := filepath.Join(contextDir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("Dockerfile not found in cwd: %s; Compose file also not found", filepath.Join(contextDir, "Dockerfile"))
}

func resolveOptionalFile(requested string) (string, error) {
	if requested == "" {
		return "", nil
	}
	path, err := filepath.Abs(requested)
	if err != nil {
		return "", fmt.Errorf("resolve env file: %w", err)
	}
	return requireRegularFile(path, "env file")
}

func requireRegularFile(path string, label string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", strings.ToLower(label), err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%s not found: %s", label, absolute)
		}
		return "", fmt.Errorf("stat %s: %w", strings.ToLower(label), err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s path is a directory: %s", label, absolute)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s path is not a regular file: %s", label, absolute)
	}
	return absolute, nil
}

func composeArgs(composePath string, contextDir string, envPath string) []string {
	args := []string{"compose", "-f", composePath, "--project-directory", contextDir}
	if envPath != "" {
		args = append(args, "--env-file", envPath)
	}
	return args
}

func composeCommand(args []string) string {
	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, "docker", "compose")
	for _, arg := range args[1:] {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func validateComposeConfig(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "docker", append(args, "config", "--quiet")...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("validate Compose config: %w", err)
	}
	return nil
}

func readComposeConfig(ctx context.Context, args []string) (composeModel, error) {
	output, err := exec.CommandContext(ctx, "docker", append(args, "config", "--format", "json")...).Output()
	if err != nil {
		return composeModel{}, fmt.Errorf("read Compose config: %w", err)
	}
	var config composeModel
	decoder := json.NewDecoder(bytes.NewReader(output))
	if err := decoder.Decode(&config); err != nil {
		return composeModel{}, fmt.Errorf("parse Compose config: %w", err)
	}
	return config, nil
}
