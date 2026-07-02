package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteShipConfigRestrictsExistingConfigFile(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.env")
	if err := os.WriteFile(configPath, []byte("CLOUDFLARE_API_TOKEN=old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SHIP_CONFIG", configPath)
	t.Setenv("SHIP_DOMAIN", "example.com")
	t.Setenv("SHIP_DNS", "cloudflare")
	t.Setenv("CLOUDFLARE_API_TOKEN", "test-token")

	if err := writeShipConfig(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
	config := readFile(t, configPath)
	for _, want := range []string{"SHIP_DOMAIN=example.com", "SHIP_DNS=cloudflare", "CLOUDFLARE_API_TOKEN=test-token"} {
		if !strings.Contains(config, want) {
			t.Fatalf("config missing %q:\n%s", want, config)
		}
	}
}
