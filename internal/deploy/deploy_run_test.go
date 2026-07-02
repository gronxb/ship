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
