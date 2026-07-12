package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDeployInitialInternetExposureRequiresExistingTailscale(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	logPath := filepath.Join(dir, "commands.log")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, filepath.Join(project, "Dockerfile"), "FROM busybox\n")
	t.Setenv("SHIP_CONFIG", filepath.Join(dir, "missing.env"))
	writeExecutable(t, filepath.Join(binDir, "docker"), "docker\n", logPath)
	writeExecutable(t, filepath.Join(binDir, "kind"), "kind\n", logPath)
	writeFile(t, filepath.Join(binDir, "kubectl"), "#!/bin/sh\nprintf 'kubectl %s\\n' \"$*\" >> "+shellQuote(logPath)+"\nexit 1\n")
	if err := os.Chmod(filepath.Join(binDir, "kubectl"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CLOUDFLARE_API_TOKEN", "token")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "account-id")
	t.Setenv("CLOUDFLARE_TUNNEL_ID", "tunnel-id")

	err := run(context.Background(), []string{"--service", "demo", "--cwd", project, "--exposure", "internet"})
	if err == nil {
		t.Fatal("expected initial internet exposure error")
	}
	if !strings.Contains(err.Error(), "internet exposure requires an existing Tailscale deployment") {
		t.Fatalf("unexpected error: %v", err)
	}
	if log := readFile(t, logPath); strings.Contains(log, "docker") || strings.Contains(log, "kind") {
		t.Fatalf("internet exposure preflight should run before deploy commands:\n%s", log)
	}
}

func TestRunDeployRejectsConfiguredDashboardInternetExposure(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.env")
	writeFile(t, configPath, "SHIP_DASHBOARD_SERVICE=ops\n")
	t.Setenv("SHIP_CONFIG", configPath)

	err := run(context.Background(), []string{"--service", "ops", "--exposure", "internet", "--dry-run"})
	if err == nil {
		t.Fatal("expected dashboard internet exposure error")
	}
	if !strings.Contains(err.Error(), "Ship dashboard cannot be exposed to the internet") {
		t.Fatalf("unexpected error: %v", err)
	}
}
func TestRunDeployDryRunPreservesExistingInternetExposure(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, filepath.Join(project, "Dockerfile"), "FROM busybox\n")
	writeFile(t, filepath.Join(binDir, "kubectl"), "#!/bin/sh\nprintf '%s' "+shellQuote(`{"metadata":{"labels":{"ship.local/exposure":"internet"}}}`)+"\n")
	if err := os.Chmod(filepath.Join(binDir, "kubectl"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"--service", "demo", "--cwd", project, "--dry-run", "--json"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output.String(), `"exposure": "internet"`) {
		t.Fatalf("deploy dry-run should preserve internet exposure:\n%s", output.String())
	}
}

func TestRunDeployDryRunPreservesExistingTailscaleExposure(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.env")
	project := filepath.Join(dir, "project")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, filepath.Join(project, "Dockerfile"), "FROM busybox\nEXPOSE 3131\n")
	writeFile(t, configPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"SHIP_DNS=cloudflare",
		"SHIP_EXPOSURE=internet",
	}, "\n")+"\n")
	t.Setenv("SHIP_CONFIG", configPath)
	writeFile(t, filepath.Join(binDir, "kubectl"), "#!/bin/sh\nprintf '%s' "+shellQuote(`{"metadata":{"labels":{"ship.local/exposure":"tailscale"}}}`)+"\n")
	if err := os.Chmod(filepath.Join(binDir, "kubectl"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"--service", "demo", "--cwd", project, "--dry-run", "--json"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output.String(), `"exposure": "tailscale"`) {
		t.Fatalf("deploy dry-run should preserve tailscale exposure:\n%s", output.String())
	}
	if !strings.Contains(output.String(), "'ship' dns publish --record 'demo.example.com'") {
		t.Fatalf("tailscale redeploy should keep service DNS reconcile:\n%s", output.String())
	}
	if strings.Contains(output.String(), "cloudflare tunnel expose demo.example.com") {
		t.Fatalf("tailscale redeploy should not plan internet tunnel exposure:\n%s", output.String())
	}
}

func TestRunDeployDryRunPlansServiceDNSReconcileWhenCloudflareDNSConfigured(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.env")
	project := filepath.Join(dir, "project")
	writeFile(t, filepath.Join(project, "Dockerfile"), "FROM busybox\nEXPOSE 3131\n")
	writeFile(t, configPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"SHIP_DNS=cloudflare",
	}, "\n")+"\n")
	t.Setenv("SHIP_CONFIG", configPath)

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"--service", "e2e", "--cwd", project, "--dry-run", "--json"}); err != nil {
			t.Fatal(err)
		}
	})

	for _, want := range []string{
		`"host": "e2e.example.com"`,
		"'ship' dns publish --record 'e2e.example.com'",
		"kubectl get gateway 'ship-tailscale' -n 'ship-system'",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("deploy dry-run missing %q:\n%s", want, output.String())
		}
	}
}

func TestRunDeployDryRunPlansServiceDNSWithoutSourceDir(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.env")
	project := filepath.Join(dir, "project")
	writeFile(t, filepath.Join(project, "Dockerfile"), "FROM busybox\nEXPOSE 3131\n")
	writeFile(t, configPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"SHIP_DNS=cloudflare",
	}, "\n")+"\n")
	t.Setenv("SHIP_CONFIG", configPath)

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"--service", "e2e", "--cwd", project, "--dry-run", "--json"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output.String(), "'ship' dns publish --record 'e2e.example.com'") {
		t.Fatalf("deploy dry-run missing service DNS publish command:\n%s", output.String())
	}
}
