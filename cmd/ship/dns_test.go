package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDNSPublishUpdatesCloudflareRecord(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.env")
	writeFile(t, configPath, strings.Join([]string{
		"SHIP_DOMAIN=example.com",
		"CLOUDFLARE_API_TOKEN=test-token",
	}, "\n")+"\n")
	t.Setenv("SHIP_CONFIG", configPath)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			w.Write([]byte(`{"success":true,"result":[{"id":"zone-id","name":"example.com"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/zones/zone-id/dns_records":
			w.Write([]byte(`{"success":true,"result":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/zones/zone-id/dns_records":
			w.Write([]byte(`{"success":true,"result":{"id":"record-id"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	withStdout(t, &output, func() {
		err := run(context.Background(), []string{
			"dns", "publish",
			"--record", "demo.example.com",
			"--target", "100.124.154.47",
			"--api-endpoint", server.URL,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output.String(), "created: demo.example.com A 100.124.154.47 proxied=false") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}
