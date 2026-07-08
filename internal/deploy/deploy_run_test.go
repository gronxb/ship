package deploy

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDryRunTextIncludesCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	err := Run(context.Background(), Options{
		ServiceName: "demo",
		CWD:         dir,
		DNSMode:     "cloudflare",
		DryRun:      true,
		ImageTag:    "test",
	}, &output)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"apiVersion: v1", "Commands:", "ship' dns publish --record 'demo.example.com'"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, output.String())
		}
	}
}

func TestRequireExistingTailscaleDeploymentAllowsTailscaleHTTPRoute(t *testing.T) {
	dir := t.TempDir()
	writeDeployExecutable(t, filepath.Join(dir, "kubectl"), "printf '%s' "+shellQuote(`{"metadata":{"labels":{"ship.local/exposure":"tailscale"}}}`)+"\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := requireExistingTailscaleDeployment(context.Background(), Result{
		ServiceName: "demo",
		Namespace:   "ship-services",
	})
	if err != nil {
		t.Fatalf("expected tailscale HTTPRoute to allow internet exposure: %v", err)
	}
}

func TestRequireExistingTailscaleDeploymentRejectsInternetHTTPRoute(t *testing.T) {
	dir := t.TempDir()
	writeDeployExecutable(t, filepath.Join(dir, "kubectl"), "printf '%s' "+shellQuote(`{"metadata":{"labels":{"ship.local/exposure":"internet"}}}`)+"\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := requireExistingTailscaleDeployment(context.Background(), Result{
		ServiceName: "demo",
		Namespace:   "ship-services",
	})
	if err == nil {
		t.Fatal("expected internet-labelled HTTPRoute to be rejected")
	}
	if !strings.Contains(err.Error(), "current exposure is internet") {
		t.Fatalf("unexpected error: %v", err)
	}
}
