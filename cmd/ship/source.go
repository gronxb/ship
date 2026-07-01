package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var sourceRepo = "gronxb/ship"
var sourceRef = "latest"

func shipSource(ctx context.Context) (string, func(), error) {
	if source := os.Getenv("SHIP_SOURCE_DIR"); source != "" {
		config := loadConfig()
		if ref := configDefault(config, "SHIP_REF", ""); ref != "" {
			_ = os.Setenv("SHIP_SOURCE_REF", ref)
		}
		return source, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "ship-source-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	config := loadConfig()
	repo := configDefault(config, "SHIP_REPO", sourceRepo)
	ref := configDefault(config, "SHIP_REF", sourceRef)
	if ref == "latest" {
		var err error
		ref, err = latestReleaseTag(ctx, repo)
		if err != nil {
			cleanup()
			return "", func() {}, fmt.Errorf("resolve latest ship release: %w", err)
		}
	}
	if err := os.Setenv("SHIP_SOURCE_REF", ref); err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := downloadSource(ctx, dir, repo, ref); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return dir, cleanup, nil
}

func downloadSource(ctx context.Context, dir string, repo string, ref string) error {
	curl := exec.CommandContext(ctx, "curl", "-fsSL", archiveURL(repo, ref))
	tar := exec.CommandContext(ctx, "tar", "-xz", "-C", dir, "--strip-components=1")
	pipe, err := curl.StdoutPipe()
	if err != nil {
		return err
	}
	curl.Stderr = os.Stderr
	tar.Stdin = pipe
	tar.Stdout = os.Stdout
	tar.Stderr = os.Stderr
	if err := curl.Start(); err != nil {
		return err
	}
	if err := tar.Start(); err != nil {
		return err
	}
	curlErr := curl.Wait()
	tarErr := tar.Wait()
	if curlErr != nil {
		return fmt.Errorf("download ship source: %w", curlErr)
	}
	if tarErr != nil {
		return fmt.Errorf("extract ship source: %w", tarErr)
	}
	return nil
}

func archiveURL(repo string, ref string) string {
	kind := "heads"
	if strings.HasPrefix(ref, "v") {
		kind = "tags"
	}
	return "https://github.com/" + repo + "/archive/refs/" + kind + "/" + ref + ".tar.gz"
}

func releaseAssetURL(repo string, ref string) (string, error) {
	osName := runtime.GOOS
	if osName != "darwin" && osName != "linux" {
		return "", fmt.Errorf("unsupported OS for release asset: %s", osName)
	}
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return "", fmt.Errorf("unsupported architecture for release asset: %s", arch)
	}
	asset := "ship_" + ref + "_" + osName + "_" + arch + ".tar.gz"
	return "https://github.com/" + repo + "/releases/download/" + ref + "/" + asset, nil
}

func latestReleaseTag(ctx context.Context, repo string) (string, error) {
	curl := exec.CommandContext(ctx, "curl", "-fsSL", "https://api.github.com/repos/"+repo+"/releases/latest")
	output, err := curl.Output()
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(output, &release); err != nil {
		return "", fmt.Errorf("parse latest release: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("latest release missing tag_name")
	}
	return release.TagName, nil
}

func runInDir(ctx context.Context, dir string, name string) error {
	cmd := exec.CommandContext(ctx, name)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s in %s: %w", name, dir, err)
	}
	return nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
