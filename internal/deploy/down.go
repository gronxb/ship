package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type DownOptions struct {
	ServiceName         string
	Namespace           string
	Domain              string
	DryRun              bool
	Registry            string
	KindCluster         string
	DNSMode             string
	CloudflareAPIKey    string
	CloudflareZoneID    string
	CloudflareAccountID string
	CloudflareTunnelID  string
	DashboardService    string
}

type deploymentImageState struct {
	Spec struct {
		Template struct {
			Spec struct {
				Containers []struct {
					Name  string `json:"name"`
					Image string `json:"image"`
				} `json:"containers"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
}

func Down(ctx context.Context, opts DownOptions, output io.Writer) error {
	opts = withDownDefaults(opts)
	if err := validateDownOptions(opts); err != nil {
		return err
	}

	exposure, err := CurrentExposure(ctx, opts.ServiceName, opts.Namespace)
	if err != nil && !errors.Is(err, ErrHTTPRouteNotFound) {
		return err
	}
	image, err := currentDeploymentImage(ctx, opts.ServiceName, opts.Namespace)
	if err != nil {
		return err
	}

	if opts.DryRun {
		printDownPlan(output, opts, exposure, image)
		return nil
	}
	if err := removeServiceCloudflareState(ctx, opts, exposure); err != nil {
		return err
	}
	if err := runCommand(ctx, "kubectl", "delete", "deployment/"+opts.ServiceName, "-n", opts.Namespace, "--ignore-not-found", "--wait=true"); err != nil {
		return err
	}
	if err := runCommand(ctx, "kubectl", "delete",
		"service/"+opts.ServiceName,
		"httproute/"+opts.ServiceName,
		"ingress/"+opts.ServiceName,
		"endpointslice/"+opts.ServiceName+"-compose",
		"secret/"+opts.ServiceName+"-env",
		"-n", opts.Namespace,
		"--ignore-not-found",
	); err != nil {
		return err
	}
	if image != "" {
		if opts.Registry == "" {
			if err := removeKindImage(ctx, opts.KindCluster, image); err != nil {
				return err
			}
		}
		if err := removeLocalDockerImage(ctx, image); err != nil {
			return err
		}
	}

	fmt.Fprintf(output, "ok: service %s is down\n", opts.ServiceName)
	if image != "" && opts.Registry != "" {
		fmt.Fprintf(output, "preserved: remote registry image %s\n", image)
	}
	return nil
}

func withDownDefaults(opts DownOptions) DownOptions {
	if opts.Namespace == "" {
		opts.Namespace = "ship-services"
	}
	if opts.Domain == "" {
		opts.Domain = "example.com"
	}
	if opts.KindCluster == "" {
		opts.KindCluster = "ship"
	}
	if opts.DNSMode == "" {
		opts.DNSMode = "manual"
	}
	return opts
}

func validateDownOptions(opts DownOptions) error {
	if opts.ServiceName == "" {
		return errors.New("--service is required")
	}
	if err := validateDNSLabel("service name", opts.ServiceName); err != nil {
		return err
	}
	if err := validateDNSLabel("namespace", opts.Namespace); err != nil {
		return err
	}
	if opts.DashboardService != "" && opts.ServiceName == opts.DashboardService {
		return fmt.Errorf("Ship dashboard cannot be brought down individually; use ship uninstall")
	}
	return nil
}

func currentDeploymentImage(ctx context.Context, serviceName string, namespace string) (string, error) {
	output, err := exec.CommandContext(ctx, "kubectl", "get", "deployment", serviceName, "-n", namespace, "--ignore-not-found", "-o", "json").Output()
	if err != nil {
		return "", fmt.Errorf("read Deployment %s/%s: %w", namespace, serviceName, err)
	}
	if len(strings.TrimSpace(string(output))) == 0 {
		return "", nil
	}
	var state deploymentImageState
	if err := json.Unmarshal(output, &state); err != nil {
		return "", fmt.Errorf("parse Deployment %s/%s: %w", namespace, serviceName, err)
	}
	for _, container := range state.Spec.Template.Spec.Containers {
		if container.Name == serviceName {
			return container.Image, nil
		}
	}
	if len(state.Spec.Template.Spec.Containers) == 1 {
		return state.Spec.Template.Spec.Containers[0].Image, nil
	}
	return "", nil
}

func removeKindImage(ctx context.Context, cluster string, image string) error {
	output, err := exec.CommandContext(ctx, "kind", "get", "nodes", "--name", cluster).Output()
	if err != nil {
		return fmt.Errorf("list kind nodes for image cleanup: %w", err)
	}
	for _, node := range strings.Fields(string(output)) {
		present, err := exec.CommandContext(ctx, "docker", "exec", node, "crictl", "inspecti", image).CombinedOutput()
		if err != nil {
			if isMissingImageOutput(present) {
				continue
			}
			return fmt.Errorf("inspect image %s on kind node %s: %w", image, node, err)
		}
		if err := runCommand(ctx, "docker", "exec", node, "crictl", "rmi", image); err != nil {
			return err
		}
	}
	return nil
}

func removeLocalDockerImage(ctx context.Context, image string) error {
	output, err := exec.CommandContext(ctx, "docker", "image", "inspect", image).CombinedOutput()
	if err != nil {
		if isMissingImageOutput(output) {
			return nil
		}
		return fmt.Errorf("inspect local Docker image %s: %w", image, err)
	}
	return runCommand(ctx, "docker", "image", "rm", image)
}

func isMissingImageOutput(output []byte) bool {
	lowered := strings.ToLower(string(output))
	return strings.Contains(lowered, "no such image") || strings.Contains(lowered, "not found")
}

func printDownPlan(output io.Writer, opts DownOptions, exposure string, image string) {
	if exposure == "internet" {
		fmt.Fprintf(output, "remove Cloudflare route for %s.%s\n", opts.ServiceName, opts.Domain)
	}
	fmt.Fprintf(output, "kubectl delete deployment/%s -n %s --ignore-not-found --wait=true\n", opts.ServiceName, opts.Namespace)
	fmt.Fprintf(output, "kubectl delete service/%s httproute/%s ingress/%s endpointslice/%s-compose secret/%s-env -n %s --ignore-not-found\n", opts.ServiceName, opts.ServiceName, opts.ServiceName, opts.ServiceName, opts.ServiceName, opts.Namespace)
	if image == "" {
		return
	}
	if opts.Registry == "" {
		fmt.Fprintf(output, "remove image %s from kind cluster %s\n", image, opts.KindCluster)
	}
	fmt.Fprintf(output, "docker image rm %s\n", image)
	if opts.Registry != "" {
		fmt.Fprintf(output, "preserve remote registry image %s\n", image)
	}
}

func removeServiceCloudflareState(ctx context.Context, opts DownOptions, exposure string) error {
	cloudflareDNS := opts.DNSMode == "cloudflare" || (opts.DNSMode == "auto" && opts.CloudflareAPIKey != "")
	if exposure != "internet" && !cloudflareDNS {
		return nil
	}
	if opts.CloudflareAPIKey == "" {
		return errors.New("missing CLOUDFLARE_API_TOKEN; refusing to remove service before DNS cleanup")
	}
	return RemoveCloudflareRoute(ctx, RemoveTunnelRouteOptions{
		Host:         opts.ServiceName + "." + opts.Domain,
		Domain:       opts.Domain,
		APIToken:     opts.CloudflareAPIKey,
		ZoneID:       opts.CloudflareZoneID,
		AccountID:    opts.CloudflareAccountID,
		TunnelID:     opts.CloudflareTunnelID,
		RemoveTunnel: exposure == "internet",
		APIEndpoint:  CloudflareAPIEndpoint,
	})
}
