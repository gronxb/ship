package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanComposeProject(t *testing.T) {
	// Given a Compose project whose gateway publishes HTTP on the host.
	dir := composeProject(t)
	envPath := filepath.Join(dir, ".env")
	writeDeployFile(t, envPath, "SUPER_SECRET=do-not-leak\n")
	setupComposePlanningTools(t, composeConfig(`{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp"}]}}`), "")

	// When Ship plans the project without a Dockerfile.
	result, err := Plan(Options{ServiceName: "demo", CWD: dir, EnvFile: envPath, DryRun: true, ImageTag: "test"})
	if err != nil {
		t.Fatal(err)
	}

	// Then it routes a selectorless Service through an EndpointSlice to Compose.
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"runtime":"compose"`, "kind: EndpointSlice", `- "0.250.250.254"`, "port: 18080", "docker compose"} {
		if !strings.Contains(string(encoded)+result.Manifest+strings.Join(result.Commands, "\n"), want) {
			t.Fatalf("Compose plan missing %q:\n%s", want, string(encoded))
		}
	}
	for _, forbidden := range []string{"kind: Deployment", "kind: Secret", "SUPER_SECRET", "do-not-leak", "docker build", "kind load"} {
		if strings.Contains(string(encoded)+result.Manifest+strings.Join(result.Commands, "\n"), forbidden) {
			t.Fatalf("Compose plan unexpectedly contains %q", forbidden)
		}
	}
	if result.Port != 18080 {
		t.Fatalf("expected published port 18080, got %d", result.Port)
	}
}

func TestPlanComposeDoesNotMountEnvFileAsKubernetesSecret(t *testing.T) {
	// Given a Compose env file containing a recognizable secret.
	dir := composeProject(t)
	envPath := filepath.Join(dir, ".env")
	writeDeployFile(t, envPath, "PASSWORD=compose-only-secret\n")
	setupComposePlanningTools(t, composeConfig(`{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp"}]}}`), "")

	// When Ship creates a dry-run plan.
	result, err := Plan(Options{ServiceName: "demo", CWD: dir, EnvFile: envPath, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}

	// Then Kubernetes never receives the Compose environment.
	all := result.Manifest + strings.Join(result.Commands, "\n")
	for _, forbidden := range []string{"kind: Secret", "kubectl create secret", "compose-only-secret"} {
		if strings.Contains(all, forbidden) {
			t.Fatalf("Compose plan unexpectedly contains %q", forbidden)
		}
	}
}

func TestPlanComposeDoesNotExposeResolvedProjectName(t *testing.T) {
	// Given a Compose project name that could have been interpolated from an env file.
	dir := composeProject(t)
	config := `{"name":"sensitiveprojectvalue","services":{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp"}]}}}`
	setupComposePlanningTools(t, config, "")

	// When Ship renders a JSON dry-run plan.
	result, err := Plan(Options{ServiceName: "demo", CWD: dir, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	// Then Compose-owned project identity stays private and Compose resolves it itself.
	if strings.Contains(string(encoded), "sensitiveprojectvalue") || strings.Contains(strings.Join(result.Commands, "\n"), " -p ") {
		t.Fatalf("Compose project identity leaked into plan: %s", encoded)
	}
}

func TestPlanCompose_rejects_missing_service(t *testing.T) {
	// Given a Compose project without any services.
	dir := composeProject(t)
	setupComposePlanningTools(t, composeConfig(`{}`), "")

	// When Ship tries to select a routable service.
	_, err := Plan(Options{ServiceName: "demo", CWD: dir, DryRun: true})

	// Then it returns an actionable service selection error.
	if err == nil || !strings.Contains(err.Error(), "Compose service") {
		t.Fatalf("expected Compose service error, got %v", err)
	}
}

func TestPlanCompose_rejects_service_without_published_http_port(t *testing.T) {
	// Given a gateway service without a published port.
	dir := composeProject(t)
	setupComposePlanningTools(t, composeConfig(`{"gateway":{}}`), "")

	// When Ship plans the route.
	_, err := Plan(Options{ServiceName: "demo", CWD: dir, DryRun: true})

	// Then it explains that a published TCP port is required.
	if err == nil || !strings.Contains(err.Error(), "published TCP port") {
		t.Fatalf("expected published TCP port error, got %v", err)
	}
}

func TestPlanCompose_rejects_loopback_binding(t *testing.T) {
	// Given a gateway port published only on host loopback.
	dir := composeProject(t)
	setupComposePlanningTools(t, composeConfig(`{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp","host_ip":"127.0.0.1"}]}}`), "")

	// When Ship plans the route.
	_, err := Plan(Options{ServiceName: "demo", CWD: dir, DryRun: true})

	// Then it rejects the unreachable binding.
	if err == nil || !strings.Contains(err.Error(), "published port is bound to loopback") {
		t.Fatalf("expected loopback binding error, got %v", err)
	}
}

func TestPlanComposeDeduplicatesDualStackPortBindings(t *testing.T) {
	// Given equivalent IPv4 and IPv6 publications for one container port.
	dir := composeProject(t)
	config := composeConfig(`{"gateway":{"ports":[{"target":443,"published":"18443","protocol":"tcp","host_ip":"0.0.0.0"},{"target":443,"published":"18443","protocol":"tcp","host_ip":"::"}]}}`)
	setupComposePlanningTools(t, config, "")

	// When the container port is selected explicitly.
	result, err := Plan(Options{ServiceName: "demo", CWD: dir, Port: 443, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}

	// Then Ship treats both host bindings as one published endpoint.
	if result.PublishedPort != 18443 {
		t.Fatalf("expected published port 18443, got %d", result.PublishedPort)
	}
}

func TestPlanComposeRejectsIPv6OnlyPublishedPort(t *testing.T) {
	// Given a publication that listens only on IPv6 while Ship routes to an IPv4 host gateway.
	dir := composeProject(t)
	config := composeConfig(`{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp","host_ip":"::"}]}}`)
	setupComposePlanningTools(t, config, "")

	// When Ship plans the external endpoint.
	_, err := Plan(Options{ServiceName: "demo", CWD: dir, DryRun: true})

	// Then it rejects the unreachable address-family mismatch.
	if err == nil || !strings.Contains(err.Error(), "IPv6-only") {
		t.Fatalf("expected IPv6-only binding error, got %v", err)
	}
}

func TestPlanComposeRejectsInterfaceSpecificPublishedPort(t *testing.T) {
	dir := composeProject(t)
	config := composeConfig(`{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp","host_ip":"192.168.1.10"}]}}`)
	setupComposePlanningTools(t, config, "")

	_, err := Plan(Options{ServiceName: "demo", CWD: dir, DryRun: true})
	if err == nil || !strings.Contains(err.Error(), "interface-specific") {
		t.Fatalf("expected interface-specific binding error, got %v", err)
	}
}

func TestPlanPrefersDockerfileWhenComposeAlsoExists(t *testing.T) {
	// Given both supported project sources.
	dir := composeProject(t)
	writeDeployFile(t, filepath.Join(dir, "Dockerfile"), "FROM busybox\nEXPOSE 3131\n")

	// When Ship plans without an explicit Compose file.
	result, err := Plan(Options{ServiceName: "demo", CWD: dir, DryRun: true, ImageTag: "test"})
	if err != nil {
		t.Fatal(err)
	}

	// Then the existing Dockerfile behavior remains the default.
	if result.Port != 3131 || !strings.Contains(strings.Join(result.Commands, "\n"), "docker build") {
		t.Fatalf("expected Dockerfile plan, got %+v", result)
	}
}

func TestPlanComposeServiceExplicitlySelectsCompose(t *testing.T) {
	// Given a project with both sources and an explicit Compose service selection.
	dir := composeProject(t)
	writeDeployFile(t, filepath.Join(dir, "Dockerfile"), "FROM busybox\nEXPOSE 3131\n")
	setupComposePlanningTools(t, composeConfig(`{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp"}]}}`), "")

	// When the caller selects the Compose gateway.
	result, err := Plan(Options{ServiceName: "demo", CWD: dir, ComposeService: "gateway", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}

	// Then the explicit selector overrides automatic Dockerfile precedence.
	if result.Runtime != "compose" {
		t.Fatalf("expected Compose runtime, got %q", result.Runtime)
	}
}

func TestRunComposePlanHonorsCanceledContext(t *testing.T) {
	// Given a Compose plan whose caller has already canceled the operation.
	dir := composeProject(t)
	setupComposePlanningTools(t, composeConfig(`{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp"}]}}`), "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// When Ship enters Compose planning.
	err := Run(ctx, Options{ServiceName: "demo", CWD: dir, DryRun: true}, io.Discard)

	// Then external planning commands honor the caller cancellation.
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func composeProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeDeployFile(t, filepath.Join(dir, "docker-compose.yml"), "services:\n  gateway:\n    image: example/gateway\n")
	return dir
}

func composeConfig(services string) string {
	return `{"name":"demo","services":` + services + `}`
}

func setupComposePlanningTools(t *testing.T, configJSON string, logPath string) {
	t.Helper()
	dir := t.TempDir()
	logCommand := ""
	if logPath != "" {
		logCommand = "printf 'docker %s\\n' \"$*\" >> " + shellQuote(logPath) + "\n"
	}
	dockerBody := logCommand + "case \"$*\" in\n" +
		"  *'config --quiet'*) exit 0 ;;\n" +
		"  *'config --format json'*) printf '%s' " + shellQuote(configJSON) + " ;;\n" +
		"  'exec ship-control-plane getent ahostsv4 host.docker.internal') printf '0.250.250.254 STREAM host.docker.internal\\n' ;;\n" +
		"esac\n"
	writeDeployExecutable(t, filepath.Join(dir, "docker"), dockerBody)
	writeDeployExecutable(t, filepath.Join(dir, "kind"), "printf 'ship-control-plane\\n'\n")
	writeDeployExecutable(t, filepath.Join(dir, "kubectl"), "case \"$*\" in\n  'config current-context') printf 'kind-ship\\n' ;;\nesac\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeDeployFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
