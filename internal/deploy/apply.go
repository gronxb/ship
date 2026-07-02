package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Apply(ctx context.Context, result Result, opts Options) error {
	if err := runCommand(ctx, "docker", "build", "-f", result.DockerfilePath, "-t", result.Image, result.ContextDir); err != nil {
		return err
	}
	if opts.Registry == "" {
		if err := runCommand(ctx, "kind", "load", "docker-image", "--name", opts.KindCluster, result.Image); err != nil {
			return err
		}
	} else if err := runCommand(ctx, "docker", "push", result.Image); err != nil {
		return err
	}
	if result.EnvFilePath != "" {
		if err := applyNamespace(ctx, result.Namespace); err != nil {
			return err
		}
		if err := applyEnvSecret(ctx, result); err != nil {
			return err
		}
	}

	kubectl := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	kubectl.Stdin = strings.NewReader(result.Manifest)
	kubectl.Stdout = os.Stdout
	kubectl.Stderr = os.Stderr
	if err := kubectl.Run(); err != nil {
		return fmt.Errorf("kubectl apply: %w", err)
	}

	if err := runCommand(ctx, "kubectl", "rollout", "status", "deployment/"+result.ServiceName, "-n", result.Namespace, "--timeout=180s"); err != nil {
		return err
	}
	if result.Exposure == "internet" {
		if err := ExposeCloudflareTunnelRoute(ctx, TunnelRouteOptions{
			Host:        result.Host,
			ServiceName: result.ServiceName,
			Namespace:   result.Namespace,
			Domain:      opts.Domain,
			APIToken:    opts.CloudflareAPIKey,
			ZoneID:      opts.CloudflareZoneID,
			AccountID:   opts.CloudflareAccountID,
			TunnelID:    opts.CloudflareTunnelID,
			Output:      os.Stdout,
			APIEndpoint: CloudflareAPIEndpoint,
		}); err != nil {
			return fmt.Errorf("publish Cloudflare Tunnel route %s: %w", result.Host, err)
		}
	} else if shouldSyncServiceDNS(opts) {
		if err := syncServiceDNS(ctx, result, opts); err != nil {
			if opts.DNSMode == "cloudflare" {
				return err
			}
			fmt.Fprintf(os.Stderr, "automatic service dns skipped: %v\n", err)
		}
	}
	fmt.Printf("ok: https://%s routes through the %s Gateway\n", result.Host, result.Exposure)
	return nil
}

func syncServiceDNS(ctx context.Context, result Result, opts Options) error {
	address, err := gatewayAddress(ctx, opts.GatewayNamespace, opts.GatewayName)
	if err != nil {
		return err
	}
	if err := publishDNSRecord(ctx, DNSRecordOptions{
		Domain:      opts.Domain,
		RecordName:  result.Host,
		Target:      strings.TrimSpace(address),
		APIToken:    opts.CloudflareAPIKey,
		ZoneID:      opts.CloudflareZoneID,
		Output:      os.Stdout,
		APIEndpoint: CloudflareAPIEndpoint,
	}); err != nil {
		return fmt.Errorf("publish service DNS %s: %w", result.Host, err)
	}
	return nil
}

func applyNamespace(ctx context.Context, namespace string) error {
	create := exec.CommandContext(ctx, "kubectl", "create", "namespace", namespace, "--dry-run=client", "-o", "yaml")
	apply := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")

	pipe, err := create.StdoutPipe()
	if err != nil {
		return fmt.Errorf("namespace pipe: %w", err)
	}
	create.Stderr = os.Stderr
	apply.Stdin = pipe
	apply.Stdout = os.Stdout
	apply.Stderr = os.Stderr

	if err := create.Start(); err != nil {
		return fmt.Errorf("kubectl create namespace: %w", err)
	}
	if err := apply.Start(); err != nil {
		return fmt.Errorf("kubectl apply namespace: %w", err)
	}
	createErr := create.Wait()
	applyErr := apply.Wait()
	if createErr != nil {
		return fmt.Errorf("kubectl create namespace: %w", createErr)
	}
	if applyErr != nil {
		return fmt.Errorf("kubectl apply namespace: %w", applyErr)
	}
	return nil
}

func applyEnvSecret(ctx context.Context, result Result) error {
	create := exec.CommandContext(ctx, "kubectl", "create", "secret", "generic", result.ServiceName+"-env", "-n", result.Namespace, "--from-env-file="+result.EnvFilePath, "--dry-run=client", "-o", "yaml")
	apply := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")

	pipe, err := create.StdoutPipe()
	if err != nil {
		return fmt.Errorf("secret pipe: %w", err)
	}
	create.Stderr = os.Stderr
	apply.Stdin = pipe
	apply.Stdout = os.Stdout
	apply.Stderr = os.Stderr

	if err := create.Start(); err != nil {
		return fmt.Errorf("kubectl create secret: %w", err)
	}
	if err := apply.Start(); err != nil {
		return fmt.Errorf("kubectl apply secret: %w", err)
	}
	createErr := create.Wait()
	applyErr := apply.Wait()
	if createErr != nil {
		return fmt.Errorf("kubectl create secret: %w", createErr)
	}
	if applyErr != nil {
		return fmt.Errorf("kubectl apply secret: %w", applyErr)
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
