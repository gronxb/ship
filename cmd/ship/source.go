package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var sourceRepo = "gronxb/ship"
var sourceRef = "main"

func shipSource(ctx context.Context) (string, func(), error) {
	if source := os.Getenv("SHIP_SOURCE_DIR"); source != "" {
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
	if err := downloadSource(ctx, dir, repo, ref); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return dir, cleanup, nil
}

func downloadSource(ctx context.Context, dir string, repo string, ref string) error {
	curl := exec.CommandContext(ctx, "curl", "-fsSL", "https://github.com/"+repo+"/archive/refs/heads/"+ref+".tar.gz")
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
