package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

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
