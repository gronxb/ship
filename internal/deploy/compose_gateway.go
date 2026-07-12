package deploy

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func resolveKindHostGatewayContext(ctx context.Context, cluster string) (string, error) {
	contextOutput, err := exec.CommandContext(ctx, "kubectl", "config", "current-context").Output()
	if err != nil {
		return "", fmt.Errorf("read kubectl context: %w", err)
	}
	expected := "kind-" + cluster
	if strings.TrimSpace(string(contextOutput)) != expected {
		return "", fmt.Errorf("Compose deployment requires kubectl context %s", expected)
	}
	nodes, err := exec.CommandContext(ctx, "kind", "get", "nodes", "--name", cluster).Output()
	if err != nil {
		return "", fmt.Errorf("list kind nodes: %w", err)
	}
	node := strings.Fields(string(nodes))
	if len(node) == 0 {
		return "", fmt.Errorf("kind cluster %s has no nodes", cluster)
	}
	output, err := exec.CommandContext(ctx, "docker", "exec", node[0], "getent", "ahostsv4", "host.docker.internal").Output()
	if err != nil {
		return "", fmt.Errorf("resolve host.docker.internal from kind node %s: %w", node[0], err)
	}
	for _, field := range strings.Fields(string(output)) {
		ip := net.ParseIP(field)
		if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("kind node %s returned no routable IPv4 for host.docker.internal", node[0])
}
