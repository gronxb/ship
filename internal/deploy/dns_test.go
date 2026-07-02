package deploy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanAddsServiceDNSCommandWhenCloudflareDNSIsConfigured(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName:      "e2e",
		CWD:              dir,
		Domain:           "example.com",
		DNSMode:          "cloudflare",
		DryRun:           true,
		GatewayName:      "ship-tailscale",
		GatewayNamespace: "ship-system",
		ImageTag:         "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	commands := strings.Join(result.Commands, "\n")
	for _, want := range []string{
		"'ship' dns publish --record 'e2e.example.com'",
		"kubectl get gateway 'ship-tailscale' -n 'ship-system'",
		"-o jsonpath='{.status.addresses[0].value}'",
	} {
		if !strings.Contains(commands, want) {
			t.Fatalf("commands missing %q:\n%s", want, commands)
		}
	}
}

func TestPlanSkipsServiceDNSCommandWhenDNSIsManual(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName: "demo",
		CWD:         dir,
		DNSMode:     "manual",
		ImageTag:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	commands := strings.Join(result.Commands, "\n")
	if strings.Contains(commands, "dns publish") {
		t.Fatalf("manual dns should not publish service record:\n%s", commands)
	}
}

func TestPlanSkipsServiceDNSCommandWhenAutoDNSHasNoCloudflareToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName: "demo",
		CWD:         dir,
		DNSMode:     "auto",
		ImageTag:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	commands := strings.Join(result.Commands, "\n")
	if strings.Contains(commands, "dns publish") {
		t.Fatalf("auto dns without Cloudflare token should not publish service record:\n%s", commands)
	}
}

func TestPlanAddsServiceDNSCommandWhenAutoDNSHasCloudflareToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName:     "demo",
		CWD:             dir,
		DNSMode:         "auto",
		CloudflareToken: true,
		ImageTag:        "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	commands := strings.Join(result.Commands, "\n")
	if !strings.Contains(commands, "'ship' dns publish --record 'demo.example.com'") {
		t.Fatalf("auto dns with token should publish service record:\n%s", commands)
	}
}

func TestPlanRejectsCloudflareDNSApplyWithoutToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Plan(Options{
		ServiceName: "demo",
		CWD:         dir,
		DNSMode:     "cloudflare",
		ImageTag:    "test",
	})
	if err == nil {
		t.Fatal("expected missing Cloudflare token error")
	}
	if !strings.Contains(err.Error(), "missing CLOUDFLARE_API_TOKEN") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanQuotesServiceDNSCommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName:      "demo",
		CWD:              dir,
		DNSMode:          "cloudflare",
		DryRun:           true,
		ShipCommand:      "/tmp/ship binary's",
		GatewayName:      "ship-tailscale",
		GatewayNamespace: "ship-system",
		ImageTag:         "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	commands := strings.Join(result.Commands, "\n")
	if !strings.Contains(commands, "'/tmp/ship binary'\"'\"'s' dns publish --record 'demo.example.com'") {
		t.Fatalf("service DNS command did not quote ship command:\n%s", commands)
	}
}

func TestApplyContinuesWhenAutoServiceDNSPublishFails(t *testing.T) {
	setupFailingServiceDNS(t)
	setupKubectl(t)

	err := Apply(context.Background(), dnsApplyResult(), Options{
		DNSMode:          "auto",
		CloudflareToken:  true,
		GatewayNamespace: "ship-system",
		GatewayName:      "ship-tailscale",
	})
	if err != nil {
		t.Fatalf("auto dns should not fail deploy: %v", err)
	}
}

func TestApplyFailsWhenCloudflareServiceDNSPublishFails(t *testing.T) {
	setupFailingServiceDNS(t)
	setupKubectl(t)

	err := Apply(context.Background(), dnsApplyResult(), Options{
		DNSMode:          "cloudflare",
		GatewayNamespace: "ship-system",
		GatewayName:      "ship-tailscale",
	})
	if err == nil {
		t.Fatal("expected cloudflare dns publish failure")
	}
	if !strings.Contains(err.Error(), "publish service DNS demo.example.com") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func dnsApplyResult() Result {
	return Result{
		ServiceName:    "demo",
		Host:           "demo.example.com",
		Image:          "ship/demo:test",
		Namespace:      "ship-services",
		DockerfilePath: "/tmp/Dockerfile",
		ContextDir:     "/tmp",
		Manifest:       "kind: List\n",
		Exposure:       "tailscale",
	}
}

func setupFailingServiceDNS(t *testing.T) {
	t.Helper()
	originalGatewayAddress := gatewayAddress
	originalPublishDNSRecord := publishDNSRecord
	gatewayAddress = func(context.Context, string, string) (string, error) {
		return "100.124.154.47", nil
	}
	publishDNSRecord = func(context.Context, DNSRecordOptions) error {
		return errors.New("dns failed")
	}
	t.Cleanup(func() {
		gatewayAddress = originalGatewayAddress
		publishDNSRecord = originalPublishDNSRecord
	})
}

func setupKubectl(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	writeDeployExecutable(t, filepath.Join(dir, "docker"), "exit 0\n")
	writeDeployExecutable(t, filepath.Join(dir, "kind"), "exit 0\n")
	writeDeployExecutable(t, filepath.Join(dir, "kubectl"), "case \"$1 $2\" in\n  'create namespace') printf 'kind: Namespace\\n' ;;\n  'create secret') printf 'kind: Secret\\n' ;;\nesac\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeDeployExecutable(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
}
