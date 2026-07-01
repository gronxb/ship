package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestArchiveURLUsesTagsForReleaseVersions(t *testing.T) {
	url := archiveURL("gronxb/ship", "v1.2.3")

	if url != "https://github.com/gronxb/ship/archive/refs/tags/v1.2.3.tar.gz" {
		t.Fatalf("unexpected archive URL: %s", url)
	}
}

func TestArchiveURLKeepsBranchesForNonVersionRefs(t *testing.T) {
	url := archiveURL("gronxb/ship", "main")

	if url != "https://github.com/gronxb/ship/archive/refs/heads/main.tar.gz" {
		t.Fatalf("unexpected archive URL: %s", url)
	}
}

func TestArchiveURLMatchesReleaseWorkflowVTags(t *testing.T) {
	url := archiveURL("gronxb/ship", "vnext")

	if url != "https://github.com/gronxb/ship/archive/refs/tags/vnext.tar.gz" {
		t.Fatalf("unexpected archive URL: %s", url)
	}
}

func TestReleaseAssetURLMatchesReleaseWorkflowArtifacts(t *testing.T) {
	url, err := releaseAssetURL("gronxb/ship", "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://github.com/gronxb/ship/releases/download/v1.2.3/ship_v1.2.3_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	if url != want {
		t.Fatalf("unexpected release asset URL: %s", url)
	}
}

func TestLatestReleaseTagFromJSONAcceptsCompactJSON(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	curlPath := filepath.Join(dir, "bin", "curl")
	writeFile(t, curlPath, strings.Join([]string{
		"#!/bin/sh",
		"printf '{\"tag_name\":\"v2.0.0\"}\\n'",
	}, "\n")+"\n")
	if err := os.Chmod(curlPath, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(curlPath))

	tag, err := latestReleaseTag(context.Background(), "gronxb/ship")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v2.0.0" {
		t.Fatalf("unexpected tag: %s", tag)
	}
}

func TestShipSourceLatestReleaseFailureDoesNotFallbackToMain(t *testing.T) {
	clearShipEnv(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bin", "curl"), strings.Join([]string{
		"#!/bin/sh",
		"echo latest release unavailable >&2",
		"exit 22",
	}, "\n")+"\n")
	if err := os.Chmod(filepath.Join(dir, "bin", "curl"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(dir, "bin"))

	_, cleanup, err := shipSource(context.Background())
	if cleanup != nil {
		cleanup()
	}
	if err == nil {
		t.Fatal("shipSource should fail when latest release cannot be resolved")
	}
	if !strings.Contains(err.Error(), "resolve latest ship release") {
		t.Fatalf("unexpected error: %v", err)
	}
}
