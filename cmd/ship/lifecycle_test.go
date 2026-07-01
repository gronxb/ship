package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

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
