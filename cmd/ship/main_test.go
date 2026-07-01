package main

import (
	"bytes"
	"context"
	"io"
	"os"
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

func TestRunInstallRunsBootstrapDomainAndDashboard(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	logPath := filepath.Join(dir, "commands.log")
	envPath := filepath.Join(dir, ".env")
	configPath := filepath.Join(dir, "config.env")
	writeFile(t, envPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"CLOUDFLARE_API_TOKEN=test-token",
		"TAILSCALE_CLIENT_ID=test-client",
		"TAILSCALE_CLIENT_SECRET=test-secret",
		"SHIP_DASHBOARD_SERVICE=ops",
	}, "\n")+"\n")
	writeExecutable(t, filepath.Join(source, "scripts", "bootstrap-kind.sh"), "bootstrap-kind\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-domain.sh"), "deploy-domain\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-dashboard.sh"), "deploy-dashboard\n", logPath)
	t.Setenv("SHIP_SOURCE_DIR", source)
	t.Setenv("SHIP_CONFIG", configPath)

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"install", "--env-file", envPath}); err != nil {
			t.Fatal(err)
		}
	})

	log := readFile(t, logPath)
	for _, want := range []string{"bootstrap-kind", "deploy-domain", "deploy-dashboard"} {
		if !strings.Contains(log, want) {
			t.Fatalf("install did not run %s; log:\n%s", want, log)
		}
	}
	config := readFile(t, configPath)
	for _, want := range []string{"SHIP_DOMAIN=example.com", "SHIP_DNS=cloudflare", "SHIP_DASHBOARD_SERVICE=ops"} {
		if !strings.Contains(config, want) {
			t.Fatalf("config missing %q:\n%s", want, config)
		}
	}
	if !strings.Contains(output.String(), "ready: ship install complete at https://ops.example.com") {
		t.Fatalf("unexpected install output:\n%s", output.String())
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

func clearShipEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"SHIP_DOMAIN",
		"CLOUDFLARE_API_TOKEN",
		"CF_API_TOKEN",
		"TAILSCALE_CLIENT_ID",
		"TAILSCALE_CLIENT_SECRET",
		"SHIP_DNS",
		"SHIP_DASHBOARD_SERVICE",
		"SHIP_IMAGE_PREFIX",
		"KIND_CLUSTER",
	} {
		t.Setenv(name, "")
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeExecutable(t *testing.T, path string, marker string, logPath string) {
	t.Helper()
	writeFile(t, path, "#!/bin/sh\nprintf '%s' "+shellQuote(marker)+" >> "+shellQuote(logPath)+"\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func withStdout(t *testing.T, writer io.Writer, run func()) {
	t.Helper()
	original := os.Stdout
	reader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = pipeWriter
	defer func() {
		os.Stdout = original
	}()

	run()

	if err := pipeWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(writer, reader); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
}
