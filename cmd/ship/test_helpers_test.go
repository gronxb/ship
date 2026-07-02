package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func clearShipEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"SHIP_DOMAIN",
		"CLOUDFLARE_API_TOKEN",
		"CF_API_TOKEN",
		"CLOUDFLARE_ACCOUNT_ID",
		"CF_ACCOUNT_ID",
		"CLOUDFLARE_TUNNEL_ID",
		"SHIP_CLOUDFLARE_TUNNEL_NAME",
		"TAILSCALE_CLIENT_ID",
		"TAILSCALE_CLIENT_SECRET",
		"SHIP_DNS",
		"SHIP_DASHBOARD_SERVICE",
		"SHIP_IMAGE_PREFIX",
		"SHIP_BIN",
		"SHIP_CONFIG",
		"SHIP_REF",
		"SHIP_REPO",
		"SHIP_SOURCE_DIR",
		"SHIP_SOURCE_REF",
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
