package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var currentOS = runtime.GOOS

func runInstall(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("ship install", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	envFile := flags.String("env-file", ".env", "environment file with SHIP_DOMAIN, Cloudflare, and Tailscale values")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if currentOS == "windows" {
		return fmt.Errorf("ship install is only supported on macOS and Linux because it bootstraps POSIX shell scripts; build the Windows binary with `go build ./cmd/ship` and use `ship --service ...` against an existing cluster")
	}

	if err := loadEnvFile(*envFile); err != nil {
		return err
	}
	domain := getenv("SHIP_DOMAIN")
	token := firstEnv("CLOUDFLARE_API_TOKEN", "CF_API_TOKEN")
	tailscaleID := firstEnv("TAILSCALE_CLIENT_ID", "TAILSCALE_OAUTH_CLIENT_ID", "TS_OAUTH_CLIENT_ID")
	tailscaleSecret := firstEnv("TAILSCALE_CLIENT_SECRET", "TAILSCALE_OAUTH_CLIENT_SECRET", "TS_OAUTH_CLIENT_SECRET")
	if domain == "" {
		return fmt.Errorf("missing SHIP_DOMAIN in %s", *envFile)
	}
	if token == "" && getenv("SHIP_DNS") != "manual" {
		return fmt.Errorf("missing CLOUDFLARE_API_TOKEN in %s; set SHIP_DNS=manual to skip DNS automation", *envFile)
	}
	if tailscaleID == "" || tailscaleSecret == "" {
		return fmt.Errorf("missing TAILSCALE_CLIENT_ID or TAILSCALE_CLIENT_SECRET in %s", *envFile)
	}
	if token != "" && getenv("SHIP_DNS") == "" {
		if err := os.Setenv("SHIP_DNS", "cloudflare"); err != nil {
			return err
		}
	}
	if err := writeShipConfig(); err != nil {
		return err
	}

	source, cleanup, err := shipSource(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := runInDir(ctx, source, "./scripts/bootstrap-kind.sh"); err != nil {
		return err
	}
	deploySystem := filepath.Join(source, "deploy-system")
	if err := runInDir(ctx, deploySystem, "./deploy-domain.sh"); err != nil {
		return err
	}
	if err := runInDir(ctx, deploySystem, "./deploy-cloudflare-tunnel.sh"); err != nil {
		return err
	}
	if err := runInDir(ctx, deploySystem, "./deploy-dashboard.sh"); err != nil {
		return err
	}
	fmt.Printf("ready: ship install complete at https://%s.%s\n", configDefault(loadConfig(), "SHIP_DASHBOARD_SERVICE", "k8s"), domain)
	return nil
}

func runUninstall(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("ship uninstall", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	envFile := flags.String("env-file", ".env", "environment file with SHIP_DOMAIN and Cloudflare values")
	dryRun := flags.Bool("dry-run", false, "print cleanup actions without deleting")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if _, err := os.Stat(*envFile); err == nil {
		if err := loadEnvFile(*envFile); err != nil {
			return err
		}
	}

	config := loadConfig()
	domain := configDefault(config, "SHIP_DOMAIN", "")
	cluster := configDefault(config, "KIND_CLUSTER", "ship")
	token := firstEnv("CLOUDFLARE_API_TOKEN", "CF_API_TOKEN")
	tailscaleID := firstEnv("TAILSCALE_CLIENT_ID", "TAILSCALE_OAUTH_CLIENT_ID", "TS_OAUTH_CLIENT_ID")
	tailscaleSecret := firstEnv("TAILSCALE_CLIENT_SECRET", "TAILSCALE_OAUTH_CLIENT_SECRET", "TS_OAUTH_CLIENT_SECRET")
	dns := configDefault(config, "SHIP_DNS", "")
	if dns == "" {
		if token != "" {
			dns = "cloudflare"
		} else {
			dns = "manual"
		}
	}
	cloudflareDNS := dns == "cloudflare"
	configPath, configDir, err := shipConfigPath()
	if err != nil {
		return err
	}
	if *dryRun {
		if cloudflareDNS && domain != "" {
			fmt.Printf("delete Cloudflare wildcard DNS for *.%s\n", domain)
		}
		fmt.Println("delete Tailscale devices named ship-tailscale* and tailscale-operator*")
		fmt.Printf("kind delete cluster --name %s\n", cluster)
		fmt.Printf("rm -rf %s\n", configDir)
		return nil
	}

	if cloudflareDNS && token == "" {
		return fmt.Errorf("missing CLOUDFLARE_API_TOKEN; refusing to uninstall before Cloudflare DNS cleanup")
	}
	source := ""
	cleanup := func() {}
	if cloudflareDNS || (tailscaleID != "" && tailscaleSecret != "") {
		var err error
		source, cleanup, err = shipSource(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
	}
	if cloudflareDNS {
		if err := runInDir(ctx, filepath.Join(source, "deploy-system"), "./delete-cloudflare-dns.sh"); err != nil {
			return err
		}
	}
	if err := runCommand(ctx, "kind", "delete", "cluster", "--name", cluster); err != nil {
		return err
	}
	if tailscaleID != "" && tailscaleSecret != "" {
		if err := runInDir(ctx, filepath.Join(source, "deploy-system"), "./delete-tailscale-devices.sh"); err != nil {
			return err
		}
	} else {
		fmt.Println("skipped: missing Tailscale OAuth credentials for device cleanup")
	}
	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("remove %s: %w", configPath, err)
	}
	fmt.Println("ok: ship system uninstalled")
	return nil
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open env file %s: %w", path, err)
	}
	defer file.Close()
	return loadKeyValue(file, func(key string, value string) error {
		if os.Getenv(key) == "" && value != "" {
			return os.Setenv(key, value)
		}
		return nil
	})
}

func loadKeyValue(reader io.Reader, set func(string, string) error) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if err := set(strings.TrimSpace(key), strings.Trim(strings.TrimSpace(value), `"'`)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func getenv(name string) string {
	return os.Getenv(name)
}
