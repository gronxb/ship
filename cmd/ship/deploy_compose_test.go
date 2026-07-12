package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDeployDryRunPlansComposeProject(t *testing.T) {
	// Given an explicit Compose file and service with a private env file.
	clearShipEnv(t)
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	composePath := filepath.Join(project, "compose.yaml")
	envPath := filepath.Join(project, ".env")
	binDir := filepath.Join(dir, "bin")
	writeFile(t, composePath, "services:\n  gateway:\n    image: example/gateway\n")
	writeFile(t, envPath, "PASSWORD=cli-secret-must-not-leak\n")
	writeFile(t, filepath.Join(binDir, "docker"), "#!/bin/sh\ncase \"$*\" in\n  *'config --quiet'*) exit 0 ;;\n  *'config --format json'*) printf '%s' '{\"name\":\"demo\",\"services\":{\"gateway\":{\"ports\":[{\"target\":80,\"published\":\"18080\",\"protocol\":\"tcp\"}]}}}' ;;\n  'exec ship-control-plane getent ahostsv4 host.docker.internal') printf '0.250.250.254 STREAM host.docker.internal\\n' ;;\nesac\n")
	writeFile(t, filepath.Join(binDir, "kind"), "#!/bin/sh\nprintf 'ship-control-plane\\n'\n")
	writeFile(t, filepath.Join(binDir, "kubectl"), "#!/bin/sh\ncase \"$*\" in\n  'config current-context') printf 'kind-ship\\n' ;;\nesac\n")
	for _, name := range []string{"docker", "kind", "kubectl"} {
		if err := os.Chmod(filepath.Join(binDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SHIP_CONFIG", filepath.Join(dir, "missing.env"))

	// When the CLI creates a JSON dry-run plan.
	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"--service", "demo", "--cwd", project, "--compose-file", composePath, "--compose-service", "gateway", "--env-file", envPath, "--dry-run", "--json"}); err != nil {
			t.Fatal(err)
		}
	})

	// Then the result identifies Compose without exposing environment values.
	for _, want := range []string{`"runtime": "compose"`, `"composeService": "gateway"`, `"publishedPort": 18080`, "kind: EndpointSlice"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("Compose dry-run missing %q:\n%s", want, output.String())
		}
	}
	for _, forbidden := range []string{"cli-secret-must-not-leak", "kind: Secret", "docker build"} {
		if strings.Contains(output.String(), forbidden) {
			t.Fatalf("Compose dry-run leaked %q:\n%s", forbidden, output.String())
		}
	}
}
