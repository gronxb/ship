package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type composeContainer struct {
	Service string `json:"Service"`
	State   string `json:"State"`
	Health  string `json:"Health"`
}

func applyCompose(ctx context.Context, result Result, opts Options) (Result, error) {
	opts = withDefaults(opts)
	if err := rejectDeploymentConflict(ctx, result); err != nil {
		return Result{}, err
	}
	args := composeArgs(result.ComposeFilePath, result.ContextDir, result.EnvFilePath)
	if err := runComposeCommand(ctx, append(args, "config", "--quiet")...); err != nil {
		return Result{}, err
	}
	if err := runComposeCommand(ctx, append(args, "up", "-d", "--wait", "--wait-timeout", "180")...); err != nil {
		return Result{}, err
	}
	output, err := exec.CommandContext(ctx, "docker", append(args, "ps", "--all", "--format", "json")...).Output()
	if err != nil {
		return Result{}, fmt.Errorf("read Compose status: %w", err)
	}
	if err := requireComposeServiceReady(output, result.ComposeService); err != nil {
		return Result{}, err
	}
	hostGateway, err := resolveKindHostGatewayContext(ctx, opts.KindCluster)
	if err != nil {
		return Result{}, err
	}
	result.HostGateway = hostGateway
	result.Manifest = composeManifestFor(opts, result.Host, hostGateway, result.PublishedPort)
	if err := applyManifest(ctx, result.Manifest); err != nil {
		return Result{}, err
	}
	return result, nil
}

func runComposeCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose command failed: %w", err)
	}
	return nil
}

func rejectDeploymentConflict(ctx context.Context, result Result) error {
	output, err := exec.CommandContext(ctx, "kubectl", "get", "deployment", result.ServiceName, "-n", result.Namespace, "--ignore-not-found", "-o", "name").Output()
	if err != nil {
		return fmt.Errorf("check Kubernetes Deployment conflict: %w", err)
	}
	if strings.TrimSpace(string(output)) != "" {
		return fmt.Errorf("Compose deployment conflicts with existing Deployment %s/%s; remove it explicitly before changing runtimes", result.Namespace, result.ServiceName)
	}
	return nil
}

func rejectComposeRuntimeConflict(ctx context.Context, result Result) error {
	output, err := exec.CommandContext(ctx, "kubectl", "get", "endpointslice", result.ServiceName+"-compose", "-n", result.Namespace, "--ignore-not-found", "-o", "name").Output()
	if err != nil {
		return fmt.Errorf("check Compose runtime conflict: %w", err)
	}
	if strings.TrimSpace(string(output)) != "" {
		return fmt.Errorf("Dockerfile deployment conflicts with existing Compose runtime %s/%s-compose; remove it explicitly before changing runtimes", result.Namespace, result.ServiceName)
	}
	return nil
}

func requireComposeServiceReady(output []byte, serviceName string) error {
	containers, err := decodeComposeContainers(output)
	if err != nil {
		return fmt.Errorf("parse Compose status: %w", err)
	}
	found := false
	for _, container := range containers {
		if container.Service != serviceName {
			continue
		}
		found = true
		if !strings.EqualFold(container.State, "running") {
			return fmt.Errorf("Compose service %s is %s, expected running", serviceName, container.State)
		}
		if strings.EqualFold(container.Health, "unhealthy") {
			return fmt.Errorf("Compose service %s is unhealthy", serviceName)
		}
	}
	if !found {
		return fmt.Errorf("Compose service %s has no container in docker compose ps", serviceName)
	}
	return nil
}

func decodeComposeContainers(output []byte) ([]composeContainer, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, errors.New("empty docker compose ps output")
	}
	if trimmed[0] == '[' {
		var containers []composeContainer
		if err := json.Unmarshal(trimmed, &containers); err != nil {
			return nil, err
		}
		return containers, nil
	}
	var containers []composeContainer
	for _, line := range bytes.Split(trimmed, []byte("\n")) {
		var container composeContainer
		if err := json.Unmarshal(line, &container); err != nil {
			return nil, err
		}
		containers = append(containers, container)
	}
	return containers, nil
}
