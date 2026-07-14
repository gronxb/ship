package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyRemovesPreviousImageAfterSuccessfulRollout(t *testing.T) {
	// Given a service with an older image deployed to a kind cluster.
	dir := t.TempDir()
	logPath := filepath.Join(dir, "commands.log")
	writeDeployExecutable(t, filepath.Join(dir, "kubectl"), "printf 'kubectl %s\\n' \"$*\" >> "+shellQuote(logPath)+"\ncase \"$*\" in\n  'get deployment demo -n ship-services --ignore-not-found -o json') printf '%s' '{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"demo\",\"image\":\"ship/demo:old\"}]}}}}' ;;\n  'apply -f -') cat >/dev/null ;;\nesac\n")
	writeDeployExecutable(t, filepath.Join(dir, "kind"), "printf 'kind %s\\n' \"$*\" >> "+shellQuote(logPath)+"\nprintf 'ship-control-plane\\n'\n")
	writeDeployExecutable(t, filepath.Join(dir, "docker"), "printf 'docker %s\\n' \"$*\" >> "+shellQuote(logPath)+"\ncase \"$*\" in\n  'exec ship-control-plane ctr -n k8s.io images ls') printf 'REF TYPE DIGEST SIZE PLATFORMS LABELS\\ndocker.io/ship/demo:old type sha256:old 1B linux/arm64 managed\\nimport-2026-07-15@sha256:wrapper-old type sha256:wrapper-old 1B linux/arm64 managed\\nimport-2026-07-15@sha256:wrapper-other type sha256:wrapper-other 1B linux/arm64 managed\\n' ;;\n  'exec ship-control-plane ctr -n k8s.io content get sha256:wrapper-old') printf '%s' '{\"manifests\":[{\"digest\":\"sha256:old\"}]}' ;;\n  'exec ship-control-plane ctr -n k8s.io content get sha256:wrapper-other') printf '%s' '{\"manifests\":[{\"digest\":\"sha256:other\"}]}' ;;\n  'image inspect ship/demo:old') printf '[]\\n' ;;\nesac\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// When a new image completes its rollout.
	err := Apply(context.Background(), Result{
		ServiceName:    "demo",
		Runtime:        "dockerfile",
		Namespace:      "ship-services",
		DockerfilePath: filepath.Join(dir, "Dockerfile"),
		ContextDir:     dir,
		Image:          "ship/demo:new",
		Manifest:       "apiVersion: apps/v1\n",
		Host:           "demo.example.com",
		Exposure:       "tailscale",
	}, Options{KindCluster: "ship"})

	// Then the previous kind and local image copies are removed after rollout.
	if err != nil {
		t.Fatal(err)
	}
	commands := readDeployFile(t, logPath)
	for _, want := range []string{
		"docker exec ship-control-plane ctr -n k8s.io images rm docker.io/ship/demo:old import-2026-07-15@sha256:wrapper-old",
		"docker image rm ship/demo:old",
	} {
		if !strings.Contains(commands, want) {
			t.Fatalf("image cleanup missing %q:\n%s", want, commands)
		}
	}
	if strings.Index(commands, "kubectl rollout status") > strings.Index(commands, "docker image rm ship/demo:old") {
		t.Fatalf("previous image was removed before rollout completed:\n%s", commands)
	}
	for _, preserved := range []string{
		"docker exec ship-control-plane crictl rmi --prune",
		"docker exec ship-control-plane ctr -n k8s.io images rm import-2026-07-15@sha256:wrapper-other",
		"docker image rm ship/demo:orphaned",
		"docker image rm ship/demo:new",
	} {
		if strings.Contains(commands, preserved) {
			t.Fatalf("cleanup removed unrelated image with %q:\n%s", preserved, commands)
		}
	}
}

func TestRemoveKindImagePreservesStateSharedByCurrentTag(t *testing.T) {
	// Given old and current tags that point to the same image digest.
	dir := t.TempDir()
	logPath := filepath.Join(dir, "commands.log")
	writeDeployExecutable(t, filepath.Join(dir, "kind"), "printf 'ship-control-plane\\n'\n")
	writeDeployExecutable(t, filepath.Join(dir, "docker"), "printf 'docker %s\\n' \"$*\" >> "+shellQuote(logPath)+"\ncase \"$*\" in\n  'exec ship-control-plane ctr -n k8s.io images ls') printf 'REF TYPE DIGEST SIZE PLATFORMS LABELS\\ndocker.io/ship/demo:old type sha256:shared 1B linux/arm64 managed\\ndocker.io/ship/demo:new type sha256:shared 1B linux/arm64 managed\\nimport-2026-07-15@sha256:wrapper-shared type sha256:wrapper-shared 1B linux/arm64 managed\\n' ;;\nesac\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// When the old deployment image is removed.
	err := removeKindImage(context.Background(), "ship", "ship/demo:old")

	// Then only the old containerd reference is deleted.
	if err != nil {
		t.Fatal(err)
	}
	commands := readDeployFile(t, logPath)
	if !strings.Contains(commands, "ctr -n k8s.io images rm docker.io/ship/demo:old") {
		t.Fatalf("old image reference was not removed:\n%s", commands)
	}
	for _, preserved := range []string{"docker.io/ship/demo:new", "images rm docker.io/ship/demo:old import-2026-07-15@sha256:wrapper-shared"} {
		if strings.Contains(commands, preserved) {
			t.Fatalf("shared current image state was removed with %q:\n%s", preserved, commands)
		}
	}
}
