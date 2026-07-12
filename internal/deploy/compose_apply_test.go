package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyComposeProjectUsesComposeAndExternalRoute(t *testing.T) {
	// Given a planned Compose project and command-capturing runtime tools.
	dir := composeProject(t)
	logPath := filepath.Join(t.TempDir(), "commands.log")
	config := composeConfig(`{"gateway":{"ports":[{"target":80,"published":"18080","protocol":"tcp"}]}}`)
	setupComposePlanningTools(t, config, logPath)
	result, err := Plan(Options{ServiceName: "demo", CWD: dir, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	writeDeployFile(t, logPath, "")
	installComposeApplyTools(t, config, logPath)

	// When Ship applies the Compose plan.
	err = Apply(context.Background(), result, Options{ServiceName: "demo", CWD: dir, Namespace: "ship-services", KindCluster: "ship"})
	if err != nil {
		t.Fatal(err)
	}

	// Then Compose reaches readiness before Kubernetes publishes the external route.
	log := readDeployFile(t, logPath)
	for _, want := range []string{"docker compose", "config --quiet", "up -d --wait --wait-timeout 180", "ps --all --format json", "kubectl apply -f -"} {
		if !strings.Contains(log, want) {
			t.Fatalf("apply log missing %q:\n%s", want, log)
		}
	}
	if strings.Index(log, "up -d --wait") > strings.Index(log, "kubectl apply -f -") {
		t.Fatalf("route was applied before Compose readiness:\n%s", log)
	}
	for _, forbidden := range []string{"docker build", "kind load", "create secret", "rollout status"} {
		if strings.Contains(log, forbidden) {
			t.Fatalf("Compose apply unexpectedly ran %q:\n%s", forbidden, log)
		}
	}
}

func TestApplyDockerfileRejectsComposeRuntimeBeforeBuild(t *testing.T) {
	// Given a Ship-managed Compose EndpointSlice with the target service name.
	dir := t.TempDir()
	logPath := filepath.Join(dir, "commands.log")
	binDir := filepath.Join(dir, "bin")
	writeDeployExecutable(t, filepath.Join(binDir, "docker"), "printf 'docker %s\\n' \"$*\" >> "+shellQuote(logPath)+"\n")
	writeDeployExecutable(t, filepath.Join(binDir, "kind"), "printf 'kind %s\\n' \"$*\" >> "+shellQuote(logPath)+"\n")
	writeDeployExecutable(t, filepath.Join(binDir, "kubectl"), "printf 'kubectl %s\\n' \"$*\" >> "+shellQuote(logPath)+"\ncase \"$*\" in\n  'get endpointslice demo-compose -n ship-services --ignore-not-found -o name') printf 'endpointslice.discovery.k8s.io/demo-compose\\n' ;;\nesac\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// When a Dockerfile deployment tries to reuse that service name.
	err := Apply(context.Background(), Result{
		ServiceName:    "demo",
		Runtime:        "dockerfile",
		Namespace:      "ship-services",
		DockerfilePath: filepath.Join(dir, "Dockerfile"),
		ContextDir:     dir,
		Image:          "ship/demo:test",
	}, Options{KindCluster: "ship"})

	// Then Ship stops before a build or mixed endpoint rollout.
	if err == nil || !strings.Contains(err.Error(), "Compose runtime") {
		t.Fatalf("expected Compose runtime conflict, got %v", err)
	}
	if log := readDeployFile(t, logPath); strings.Contains(log, "docker ") || strings.Contains(log, "kind ") {
		t.Fatalf("runtime conflict should stop before build/load:\n%s", log)
	}
}

func TestApplyComposeRejectsDeploymentConflictBeforeUp(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "commands.log")
	binDir := filepath.Join(dir, "bin")
	writeDeployExecutable(t, filepath.Join(binDir, "docker"), "printf 'docker %s\\n' \"$*\" >> "+shellQuote(logPath)+"\n")
	writeDeployExecutable(t, filepath.Join(binDir, "kubectl"), "case \"$*\" in\n  'get deployment demo -n ship-services --ignore-not-found -o name') printf 'deployment.apps/demo\\n' ;;\nesac\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := Apply(context.Background(), Result{
		ServiceName:     "demo",
		Runtime:         "compose",
		Namespace:       "ship-services",
		ComposeFilePath: filepath.Join(dir, "compose.yaml"),
		ContextDir:      dir,
	}, Options{KindCluster: "ship"})
	if err == nil || !strings.Contains(err.Error(), "existing Deployment") {
		t.Fatalf("expected Deployment conflict, got %v", err)
	}
	if log, readErr := os.ReadFile(logPath); readErr == nil && strings.Contains(string(log), "docker ") {
		t.Fatalf("Deployment conflict should stop before Compose mutation:\n%s", log)
	}
}

func TestRunComposeCommandDoesNotExposeStderr(t *testing.T) {
	dir := t.TempDir()
	writeDeployExecutable(t, filepath.Join(dir, "docker"), "printf 'synthetic-secret-value\\n' >&2\nexit 1\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := runComposeCommand(context.Background(), "compose", "config", "--quiet")
	if err == nil {
		t.Fatal("expected Compose command failure")
	}
	if strings.Contains(err.Error(), "synthetic-secret-value") {
		t.Fatalf("Compose stderr leaked through error: %v", err)
	}
}

func TestRequireComposeServiceReadyFormats(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		service string
		wantErr string
	}{
		{name: "array", output: `[{"Service":"gateway","State":"running","Health":"healthy"}]`, service: "gateway"},
		{name: "unhealthy", output: `{"Service":"gateway","State":"running","Health":"unhealthy"}`, service: "gateway", wantErr: "unhealthy"},
		{name: "stopped", output: `{"Service":"gateway","State":"exited","Health":""}`, service: "gateway", wantErr: "expected running"},
		{name: "missing", output: `{"Service":"api","State":"running","Health":"healthy"}`, service: "gateway", wantErr: "has no container"},
		{name: "malformed", output: `{`, service: "gateway", wantErr: "parse Compose status"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := requireComposeServiceReady([]byte(test.output), test.service)
			if test.wantErr == "" && err != nil {
				t.Fatalf("expected ready service, got %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func installComposeApplyTools(t *testing.T, configJSON string, logPath string) {
	t.Helper()
	dir := t.TempDir()
	dockerBody := "printf 'docker %s\\n' \"$*\" >> " + shellQuote(logPath) + "\n" +
		"case \"$*\" in\n" +
		"  *'config --quiet'*) exit 0 ;;\n" +
		"  *'config --format json'*) printf '%s' " + shellQuote(configJSON) + " ;;\n" +
		"  *'ps --all --format json'*) printf '%s\\n' '{\"Service\":\"gateway\",\"State\":\"running\",\"Health\":\"healthy\"}' ;;\n" +
		"  'exec ship-control-plane getent ahostsv4 host.docker.internal') printf '0.250.250.254 STREAM host.docker.internal\\n' ;;\n" +
		"esac\n"
	writeDeployExecutable(t, filepath.Join(dir, "docker"), dockerBody)
	writeDeployExecutable(t, filepath.Join(dir, "kind"), "printf 'ship-control-plane\\n'\n")
	kubectlBody := "printf 'kubectl %s\\n' \"$*\" >> " + shellQuote(logPath) + "\n" +
		"case \"$*\" in\n" +
		"  'config current-context') printf 'kind-ship\\n' ;;\n" +
		"  'get deployment demo -n ship-services -o name') exit 1 ;;\n" +
		"  'apply -f -') cat >/dev/null ;;\n" +
		"esac\n"
	writeDeployExecutable(t, filepath.Join(dir, "kubectl"), kubectlBody)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func readDeployFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
