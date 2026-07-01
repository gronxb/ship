package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runUpgrade(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("ship upgrade", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	envFile := flags.String("env-file", ".env", "optional environment file with Ship infrastructure values")
	yes := flags.Bool("y", false, "update infrastructure without prompting")
	flags.BoolVar(yes, "yes", false, "update infrastructure without prompting")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if currentOS == "windows" {
		return fmt.Errorf("ship upgrade is only supported on macOS and Linux because infrastructure updates use POSIX shell scripts")
	}
	if _, err := os.Stat(*envFile); err == nil {
		if err := loadEnvFile(*envFile); err != nil {
			return err
		}
	}

	source, cleanup, err := shipSource(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	binPath, err := upgradeCLI(ctx, source)
	if err != nil {
		return err
	}
	fmt.Printf("upgraded: %s\n", binPath)

	if !*yes && !confirmInfrastructureUpdate(os.Stdin) {
		fmt.Println("skipped: ship infrastructure update")
		return nil
	}
	if err := os.Setenv("SHIP_BIN", binPath); err != nil {
		return err
	}
	if err := updateInfrastructure(ctx, source); err != nil {
		return err
	}
	fmt.Println("updated: ship infrastructure")
	return nil
}

func upgradeCLI(ctx context.Context, source string) (string, error) {
	binPath := os.Getenv("SHIP_BIN")
	if binPath == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("missing HOME for ship binary path")
		}
		binPath = filepath.Join(home, ".local", "bin", "ship")
	}
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		return "", err
	}
	config := loadConfig()
	ldflags := "-X main.sourceRepo=" + configDefault(config, "SHIP_REPO", sourceRepo) + " -X main.sourceRef=" + configDefault(config, "SHIP_REF", sourceRef)
	cmd := exec.CommandContext(ctx, "go", "build", "-ldflags", ldflags, "-o", binPath, "./cmd/ship")
	cmd.Dir = source
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build ship CLI: %w", err)
	}
	return binPath, nil
}

func confirmInfrastructureUpdate(input *os.File) bool {
	fmt.Print("Infrastructure may have changed. Update Ship infrastructure now? [y/N]: ")
	answer, err := bufio.NewReader(input).ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func updateInfrastructure(ctx context.Context, source string) error {
	deploySystem := filepath.Join(source, "deploy-system")
	if err := runInDir(ctx, deploySystem, "./deploy-domain.sh"); err != nil {
		return err
	}
	return runInDir(ctx, deploySystem, "./deploy-dashboard.sh")
}
