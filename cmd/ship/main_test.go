package main

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsVersionWhenShortVersionFlagIsUsed(t *testing.T) {
	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"-v"}); err != nil {
			t.Fatal(err)
		}
	})

	if strings.TrimSpace(output.String()) != "ship dev" {
		t.Fatalf("unexpected version output %q", output.String())
	}
}

func TestRunPrintsInjectedVersionWhenBuiltFromReleaseTag(t *testing.T) {
	original := version
	version = "v1.2.3"
	t.Cleanup(func() { version = original })

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"-v"}); err != nil {
			t.Fatal(err)
		}
	})

	if strings.TrimSpace(output.String()) != "ship v1.2.3" {
		t.Fatalf("unexpected version output %q", output.String())
	}
}

func TestRunPrintsRichHelpWhenLongHelpFlagIsUsed(t *testing.T) {
	// Given
	var output bytes.Buffer

	// When
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"--help"}); err != nil {
			t.Fatal(err)
		}
	})

	// Then
	for _, want := range []string{"Usage", "Commands", "Options", "down", "install", "upgrade", "uninstall", "dns publish", "--service", "--dry-run", "--version"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("help output missing %q:\n%s", want, output.String())
		}
	}
}

func TestRunDownRequiresServiceName(t *testing.T) {
	// Given no service name.
	clearShipEnv(t)

	// When down is invoked.
	err := run(context.Background(), []string{"down", "--dry-run"})

	// Then the CLI rejects the request at its boundary.
	if err == nil || !strings.Contains(err.Error(), "--service is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUninstallDryRunPrintsCleanupPlan(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	writeFile(t, envPath, "SHIP_DOMAIN=example.com\nCLOUDFLARE_API_TOKEN=test-token\n")

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"uninstall", "--env-file", envPath, "--dry-run"}); err != nil {
			t.Fatal(err)
		}
	})

	for _, want := range []string{
		"delete Cloudflare wildcard DNS for *.example.com",
		"kind delete cluster --name ship",
		"rm -rf",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("uninstall dry-run missing %q:\n%s", want, output.String())
		}
	}
}

func TestRunUninstallRequiresTokenForCloudflareCleanup(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.env")
	writeFile(t, configPath, "SHIP_DOMAIN=example.com\nSHIP_DNS=cloudflare\n")
	t.Setenv("SHIP_CONFIG", configPath)

	err := run(context.Background(), []string{"uninstall"})
	if err == nil {
		t.Fatal("expected missing Cloudflare token error")
	}
	if !strings.Contains(err.Error(), "refusing to uninstall before Cloudflare DNS cleanup") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUninstallPreservesManualDNSOwnership(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	logPath := filepath.Join(dir, "commands.log")
	envPath := filepath.Join(dir, ".env")
	configPath := filepath.Join(dir, "config.env")
	writeFile(t, envPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"SHIP_DNS=manual",
		"TAILSCALE_CLIENT_ID=test-client",
		"TAILSCALE_CLIENT_SECRET=test-secret",
	}, "\n")+"\n")
	writeExecutable(t, filepath.Join(source, "scripts", "bootstrap-kind.sh"), "bootstrap-kind\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-domain.sh"), "deploy-domain\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-cloudflare-tunnel.sh"), "deploy-cloudflare-tunnel\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-dashboard.sh"), "deploy-dashboard\n", logPath)
	t.Setenv("SHIP_SOURCE_DIR", source)
	t.Setenv("SHIP_CONFIG", configPath)

	withStdout(t, io.Discard, func() {
		if err := run(context.Background(), []string{"install", "--env-file", envPath}); err != nil {
			t.Fatal(err)
		}
	})

	config := readFile(t, configPath)
	if !strings.Contains(config, "SHIP_DNS=manual") {
		t.Fatalf("manual DNS ownership was not preserved:\n%s", config)
	}
	t.Setenv("CLOUDFLARE_API_TOKEN", "late-token")

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"uninstall", "--dry-run"}); err != nil {
			t.Fatal(err)
		}
	})

	if strings.Contains(output.String(), "delete Cloudflare wildcard DNS") {
		t.Fatalf("manual DNS uninstall should not plan Cloudflare cleanup:\n%s", output.String())
	}
}

func TestRunInstallDeploysCloudflareTunnelBeforeDashboard(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	logPath := filepath.Join(dir, "commands.log")
	envPath := filepath.Join(dir, ".env")
	configPath := filepath.Join(dir, "config.env")
	writeFile(t, envPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"CLOUDFLARE_API_TOKEN=test-token",
		"CLOUDFLARE_ACCOUNT_ID=account-id",
		"TAILSCALE_CLIENT_ID=test-client",
		"TAILSCALE_CLIENT_SECRET=test-secret",
	}, "\n")+"\n")
	writeExecutable(t, filepath.Join(source, "scripts", "bootstrap-kind.sh"), "bootstrap-kind\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-domain.sh"), "deploy-domain\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-cloudflare-tunnel.sh"), "deploy-cloudflare-tunnel\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-dashboard.sh"), "deploy-dashboard\n", logPath)
	t.Setenv("SHIP_SOURCE_DIR", source)
	t.Setenv("SHIP_CONFIG", configPath)

	withStdout(t, io.Discard, func() {
		if err := run(context.Background(), []string{"install", "--env-file", envPath}); err != nil {
			t.Fatal(err)
		}
	})

	if got := readFile(t, logPath); got != strings.Join([]string{
		"bootstrap-kind",
		"deploy-domain",
		"deploy-cloudflare-tunnel",
		"deploy-dashboard",
	}, "\n")+"\n" {
		t.Fatalf("unexpected install command order:\n%s", got)
	}
	config := readFile(t, configPath)
	if !strings.Contains(config, "CLOUDFLARE_ACCOUNT_ID=account-id") {
		t.Fatalf("config missing Cloudflare account id:\n%s", config)
	}
}

func TestShipConfigPathUsesWindowsConfigHomeWhenHomeIsMissing(t *testing.T) {
	clearShipEnv(t)
	originalOS := currentOS
	currentOS = "windows"
	t.Cleanup(func() { currentOS = originalOS })
	dir := t.TempDir()
	t.Setenv("SHIP_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "LocalAppData"))
	t.Setenv("APPDATA", filepath.Join(dir, "AppData", "Roaming"))

	path, configDir, err := shipConfigPath()
	if err != nil {
		t.Fatal(err)
	}

	wantDir := filepath.Join(dir, "LocalAppData", "ship")
	if configDir != wantDir {
		t.Fatalf("unexpected config dir %q, want %q", configDir, wantDir)
	}
	if path != filepath.Join(wantDir, "config.env") {
		t.Fatalf("unexpected config path %q", path)
	}
}

func TestRunInstallRejectsWindowsShellLifecycle(t *testing.T) {
	clearShipEnv(t)
	originalOS := currentOS
	currentOS = "windows"
	t.Cleanup(func() { currentOS = originalOS })
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	writeFile(t, envPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"CLOUDFLARE_API_TOKEN=test-token",
		"TAILSCALE_CLIENT_ID=test-client",
		"TAILSCALE_CLIENT_SECRET=test-secret",
	}, "\n")+"\n")
	t.Setenv("SHIP_CONFIG", filepath.Join(dir, "config.env"))

	err := run(context.Background(), []string{"install", "--env-file", envPath})
	if err == nil {
		t.Fatal("expected Windows install error")
	}
	if !strings.Contains(err.Error(), "ship install is only supported on macOS and Linux") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
