package deploy

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownRemovesDockerfileServiceAndLocalImages(t *testing.T) {
	// Given a Dockerfile service deployed to a two-node kind cluster.
	dir := t.TempDir()
	logPath := filepath.Join(dir, "commands.log")
	writeDeployExecutable(t, filepath.Join(dir, "kubectl"), "printf 'kubectl %s\\n' \"$*\" >> "+shellQuote(logPath)+"\ncase \"$*\" in\n  'get httproute demo -n ship-services -o json') printf '%s' '{\"metadata\":{\"labels\":{\"ship.local/exposure\":\"tailscale\"}}}' ;;\n  'get deployment demo -n ship-services --ignore-not-found -o json') printf '%s' '{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"demo\",\"image\":\"ship/demo:test\"}]}}}}' ;;\nesac\n")
	writeDeployExecutable(t, filepath.Join(dir, "kind"), "printf 'kind %s\\n' \"$*\" >> "+shellQuote(logPath)+"\nprintf 'ship-control-plane\\nship-worker\\n'\n")
	writeDeployExecutable(t, filepath.Join(dir, "docker"), "printf 'docker %s\\n' \"$*\" >> "+shellQuote(logPath)+"\ncase \"$*\" in\n  'exec '*' crictl inspecti ship/demo:test') printf '{}\\n' ;;\n  'image inspect ship/demo:test') printf '[]\\n' ;;\nesac\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// When the service is brought down.
	var output bytes.Buffer
	err := Down(context.Background(), DownOptions{
		ServiceName: "demo",
		Namespace:   "ship-services",
		KindCluster: "ship",
	}, &output)

	// Then its resources and every local image copy are removed.
	if err != nil {
		t.Fatal(err)
	}
	commands := readDeployFile(t, logPath)
	for _, want := range []string{
		"kubectl delete deployment/demo -n ship-services --ignore-not-found --wait=true",
		"kubectl delete service/demo httproute/demo ingress/demo endpointslice/demo-compose secret/demo-env -n ship-services --ignore-not-found",
		"docker exec ship-control-plane crictl rmi ship/demo:test",
		"docker exec ship-worker crictl rmi ship/demo:test",
		"docker image rm ship/demo:test",
	} {
		if !strings.Contains(commands, want) {
			t.Fatalf("down commands missing %q:\n%s", want, commands)
		}
	}
	if !strings.Contains(output.String(), "ok: service demo is down") {
		t.Fatalf("unexpected down output: %s", output.String())
	}
}

func TestDownDryRunDoesNotDeleteResourcesOrImages(t *testing.T) {
	// Given a deployed Dockerfile service.
	dir := t.TempDir()
	logPath := filepath.Join(dir, "commands.log")
	writeDeployExecutable(t, filepath.Join(dir, "kubectl"), "printf 'kubectl %s\\n' \"$*\" >> "+shellQuote(logPath)+"\ncase \"$*\" in\n  'get httproute demo -n ship-services -o json') printf '%s' '{\"metadata\":{\"labels\":{\"ship.local/exposure\":\"tailscale\"}}}' ;;\n  'get deployment demo -n ship-services --ignore-not-found -o json') printf '%s' '{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"demo\",\"image\":\"ship/demo:test\"}]}}}}' ;;\nesac\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// When down is previewed.
	var output bytes.Buffer
	err := Down(context.Background(), DownOptions{
		ServiceName: "demo",
		Namespace:   "ship-services",
		KindCluster: "ship",
		DryRun:      true,
	}, &output)

	// Then only discovery runs and the destructive plan is printed.
	if err != nil {
		t.Fatal(err)
	}
	commands := readDeployFile(t, logPath)
	if strings.Contains(commands, "kubectl delete") || strings.Contains(commands, "docker image rm") {
		t.Fatalf("dry-run executed destructive commands:\n%s", commands)
	}
	for _, want := range []string{"kubectl delete deployment/demo", "remove image ship/demo:test from kind cluster ship", "docker image rm ship/demo:test"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, output.String())
		}
	}
}
