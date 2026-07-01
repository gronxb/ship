package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
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
	t.Setenv("SHIP_REF", "v9.8.7")

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
	upgraded := filepath.Join(home, ".local", "bin", "ship")
	versionOutput := runShipVersion(t, upgraded)
	if strings.TrimSpace(versionOutput) != "ship v9.8.7" {
		t.Fatalf("upgrade did not preserve source tag version, got %q", versionOutput)
	}
}

func TestRunUpgradeYesUsesConfigRefForLocalSourceVersion(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	source := filepath.Join(dir, "source")
	logPath := filepath.Join(dir, "commands.log")
	configPath := filepath.Join(dir, "config.env")
	writeUpgradeSource(t, source, logPath)
	writeFile(t, configPath, "SHIP_REF=v7.7.7\n")
	t.Setenv("HOME", home)
	t.Setenv("SHIP_SOURCE_DIR", source)
	t.Setenv("SHIP_CONFIG", configPath)

	withStdout(t, io.Discard, func() {
		if err := run(context.Background(), []string{"upgrade", "-y"}); err != nil {
			t.Fatal(err)
		}
	})

	versionOutput := runShipVersion(t, filepath.Join(home, ".local", "bin", "ship"))
	if strings.TrimSpace(versionOutput) != "ship v7.7.7" {
		t.Fatalf("upgrade did not use config source tag version, got %q", versionOutput)
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

func TestRunUpgradeYesDefaultsToLatestReleaseTag(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	source := filepath.Join(dir, "source")
	curlLog := filepath.Join(dir, "curl.log")
	commandLog := filepath.Join(dir, "commands.log")
	archivePath := filepath.Join(dir, "v9.8.7.tar.gz")
	configPath := filepath.Join(dir, "config.env")
	writeUpgradeSource(t, source, commandLog)
	if err := exec.Command("tar", "-czf", archivePath, "-C", source, ".").Run(); err != nil {
		t.Fatal(err)
	}
	writeFakeCurl(t, filepath.Join(dir, "bin"), archivePath, curlLog)
	t.Setenv("PATH", filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HOME", home)
	t.Setenv("SHIP_CONFIG", configPath)

	withStdout(t, io.Discard, func() {
		if err := run(context.Background(), []string{"upgrade", "-y"}); err != nil {
			t.Fatal(err)
		}
	})

	curl := readFile(t, curlLog)
	for _, want := range []string{"api.github.com/repos/gronxb/ship/releases/latest", "archive/refs/tags/v9.8.7.tar.gz"} {
		if !strings.Contains(curl, want) {
			t.Fatalf("latest upgrade did not request %q; curl log:\n%s", want, curl)
		}
	}
	versionOutput := runShipVersion(t, filepath.Join(home, ".local", "bin", "ship"))
	if strings.TrimSpace(versionOutput) != "ship v9.8.7" {
		t.Fatalf("upgrade did not install latest release version, got %q", versionOutput)
	}
	log := readFile(t, commandLog)
	for _, want := range []string{"deploy-domain", "deploy-dashboard"} {
		if !strings.Contains(log, want) {
			t.Fatalf("upgrade did not run %s; log:\n%s", want, log)
		}
	}
}

func writeUpgradeSource(t *testing.T, source string, logPath string) {
	t.Helper()
	writeFile(t, filepath.Join(source, "go.mod"), "module example.com/fakeship\n\ngo 1.22\n")
	writeFile(t, filepath.Join(source, "cmd", "ship", "main.go"), "package main\n\nimport \"fmt\"\n\nvar version = \"dev\"\n\nfunc main() { fmt.Println(\"ship \" + version) }\n")
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-domain.sh"), "deploy-domain\n", logPath)
	writeExecutable(t, filepath.Join(source, "deploy-system", "deploy-dashboard.sh"), "deploy-dashboard\n", logPath)
}

func writeFakeCurl(t *testing.T, dir string, archivePath string, logPath string) {
	t.Helper()
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf 'curl %s\\n' \"$*\" >> " + shellQuote(logPath),
		"case \"$*\" in",
		"  *api.github.com*) printf '{\"tag_name\":\"v9.8.7\"}\\n' ;;",
		"  *refs/tags/v9.8.7.tar.gz*) cat " + shellQuote(archivePath) + " ;;",
		"  *) echo \"unexpected curl: $*\" >&2; exit 2 ;;",
		"esac",
	}, "\n") + "\n"
	writeFile(t, filepath.Join(dir, "curl"), script)
	if err := os.Chmod(filepath.Join(dir, "curl"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func runShipVersion(t *testing.T, binPath string) string {
	t.Helper()
	output, err := exec.Command(binPath, "-v").CombinedOutput()
	if err != nil {
		t.Fatalf("run upgraded ship -v: %v\n%s", err, output)
	}
	return string(output)
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
