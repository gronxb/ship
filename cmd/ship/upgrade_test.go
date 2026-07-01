package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUpgradeYesRebuildsCLIAndUpdatesInfrastructure(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	source := filepath.Join(dir, "source")
	logPath := filepath.Join(dir, "commands.log")
	envPath := filepath.Join(dir, ".env")
	configPath := filepath.Join(dir, "config.env")
	writeUpgradeSource(t, source, logPath)
	writeFile(t, envPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"CLOUDFLARE_API_TOKEN=test-token",
		"TAILSCALE_CLIENT_ID=test-client",
		"TAILSCALE_CLIENT_SECRET=test-secret",
	}, "\n")+"\n")
	t.Setenv("HOME", home)
	t.Setenv("SHIP_SOURCE_DIR", source)
	t.Setenv("SHIP_CONFIG", configPath)

	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"upgrade", "--env-file", envPath, "-y"}); err != nil {
			t.Fatal(err)
		}
	})

	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "ship")); err != nil {
		t.Fatalf("upgrade did not install CLI: %v", err)
	}
	log := readFile(t, logPath)
	for _, want := range []string{"deploy-domain", "deploy-dashboard"} {
		if !strings.Contains(log, want) {
			t.Fatalf("upgrade did not run %s; log:\n%s", want, log)
		}
	}
	if !strings.Contains(output.String(), "upgraded:") || !strings.Contains(output.String(), "updated: ship infrastructure") {
		t.Fatalf("unexpected upgrade output:\n%s", output.String())
	}
}

func TestRunUpgradePromptsBeforeInfrastructureUpdate(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	source := filepath.Join(dir, "source")
	logPath := filepath.Join(dir, "commands.log")
	envPath := filepath.Join(dir, ".env")
	configPath := filepath.Join(dir, "config.env")
	writeUpgradeSource(t, source, logPath)
	writeFile(t, envPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"CLOUDFLARE_API_TOKEN=test-token",
		"TAILSCALE_CLIENT_ID=test-client",
		"TAILSCALE_CLIENT_SECRET=test-secret",
	}, "\n")+"\n")
	t.Setenv("HOME", home)
	t.Setenv("SHIP_SOURCE_DIR", source)
	t.Setenv("SHIP_CONFIG", configPath)

	var output bytes.Buffer
	withStdout(t, &output, func() {
		withStdin(t, "n\n", func() {
			if err := run(context.Background(), []string{"upgrade", "--env-file", envPath}); err != nil {
				t.Fatal(err)
			}
		})
	})

	if log, err := os.ReadFile(logPath); err == nil && strings.TrimSpace(string(log)) != "" {
		t.Fatalf("upgrade should not update infrastructure after no prompt; log:\n%s", log)
	}
	if !strings.Contains(output.String(), "Update Ship infrastructure now?") || !strings.Contains(output.String(), "skipped: ship infrastructure update") {
		t.Fatalf("unexpected upgrade prompt output:\n%s", output.String())
	}
}

func writeUpgradeSource(t *testing.T, source string, logPath string) {
	t.Helper()
	writeFile(t, filepath.Join(source, "go.mod"), "module example.com/fakeship\n\ngo 1.23\n")
	writeFile(t, filepath.Join(source, "cmd", "ship", "main.go"), "package main\n\nfunc main() {}\n")
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-domain.sh"), "deploy-domain\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-dashboard.sh"), "deploy-dashboard\n", logPath)
}

func withStdin(t *testing.T, input string, run func()) {
	t.Helper()
	original := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = reader
	defer func() {
		os.Stdin = original
	}()
	if _, err := writer.WriteString(input); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	run()
}
