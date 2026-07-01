package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/gronxb/ship/internal/deploy"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			fmt.Print(`Usage:
  ship [options]
  ship install [options]
  ship upgrade [options]
  ship uninstall [options]

Commands:
  install     bootstrap ship infrastructure
  upgrade     update ship CLI and optionally infrastructure
  uninstall   remove ship infrastructure

Options:
  --service <name>   DNS-safe service name
  --dry-run          print the plan without applying it
  -v, --version      print version
  -h, --help         print help
`)
			return nil
		case "-v", "--version", "version":
			fmt.Println("ship " + version)
			return nil
		case "install":
			return runInstall(ctx, args[1:])
		case "upgrade":
			return runUpgrade(ctx, args[1:])
		case "uninstall":
			return runUninstall(ctx, args[1:])
		}
	}
	return runDeploy(ctx, args)
}

func runDeploy(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("ship", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	config := loadConfig()

	opts := deploy.Options{}
	flags.StringVar(&opts.ServiceName, "service", "", "DNS-safe service name")
	flags.StringVar(&opts.CWD, "cwd", "", "project directory containing Dockerfile")
	flags.IntVar(&opts.Port, "port", 0, "container HTTP port; defaults to Dockerfile EXPOSE or 8080")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "print the deployment plan without applying it")
	flags.BoolVar(&opts.JSON, "json", false, "print JSON output")
	flags.StringVar(&opts.Namespace, "namespace", configDefault(config, "SHIP_NAMESPACE", "ship-services"), "target Kubernetes namespace")
	flags.StringVar(&opts.Domain, "domain", configDefault(config, "SHIP_DOMAIN", "example.com"), "base DNS domain")
	flags.StringVar(&opts.GatewayNamespace, "gateway-namespace", configDefault(config, "SHIP_GATEWAY_NAMESPACE", "ship-system"), "Gateway namespace")
	flags.StringVar(&opts.GatewayName, "gateway-name", configDefault(config, "SHIP_GATEWAY_NAME", "ship-tailscale"), "Gateway name")
	flags.StringVar(&opts.InternetGateway, "internet-gateway-name", configDefault(config, "SHIP_INTERNET_GATEWAY_NAME", "ship-internet"), "Internet Gateway name")
	flags.StringVar(&opts.Exposure, "exposure", configDefault(config, "SHIP_EXPOSURE", "tailscale"), "network exposure: tailscale or internet")
	flags.StringVar(&opts.ImagePrefix, "image-prefix", configDefault(config, "SHIP_IMAGE_PREFIX", "ship"), "image name prefix")
	flags.StringVar(&opts.Registry, "registry", configDefault(config, "REGISTRY", ""), "optional image registry")
	flags.StringVar(&opts.ImageTag, "image-tag", configDefault(config, "IMAGE_TAG", ""), "image tag")
	flags.StringVar(&opts.KindCluster, "kind-cluster", configDefault(config, "KIND_CLUSTER", "ship"), "kind cluster name")
	flags.StringVar(&opts.EnvFile, "env-file", "", "optional env file to mount as a Kubernetes Secret")
	flags.StringVar(&opts.ServiceAccount, "service-account", "", "optional Kubernetes ServiceAccount name for the Deployment")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	return deploy.Run(ctx, opts, os.Stdout)
}

func configDefault(config map[string]string, name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	if value := config[name]; value != "" {
		return value
	}
	return fallback
}

func loadConfig() map[string]string {
	path := os.Getenv("SHIP_CONFIG")
	if path == "" {
		configPath, _, err := shipConfigPath()
		if err != nil {
			return map[string]string{}
		}
		path = configPath
	}

	file, err := os.Open(path)
	if err != nil {
		return map[string]string{}
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values
}
