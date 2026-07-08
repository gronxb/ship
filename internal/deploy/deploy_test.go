package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanRequiresDockerfileInCWD(t *testing.T) {
	dir := t.TempDir()

	_, err := Plan(Options{ServiceName: "demo", CWD: dir})
	if err == nil {
		t.Fatal("expected missing Dockerfile error")
	}
	if !strings.Contains(err.Error(), "Dockerfile not found in cwd") {
		t.Fatalf("expected Dockerfile error, got %v", err)
	}
}

func TestPlanBuildsTailscaleOnlyRoute(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName: "demo",
		CWD:         dir,
		Port:        3131,
		ImageTag:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Host != "demo.example.com" {
		t.Fatalf("unexpected host %q", result.Host)
	}
	if result.Image != "ship/demo:test" {
		t.Fatalf("unexpected image %q", result.Image)
	}
	if result.Exposure != "tailscale" || !result.TailscaleOnly {
		t.Fatalf("unexpected exposure %q tailscaleOnly=%t", result.Exposure, result.TailscaleOnly)
	}
	for _, want := range []string{
		`ship.local/exposure: "tailscale"`,
		`ship.local/tailscale-only: "true"`,
		"hostnames:\n    - demo.example.com",
		"containerPort: 3131",
		"name: ship-tailscale",
	} {
		if !strings.Contains(result.Manifest, want) {
			t.Fatalf("manifest missing %q:\n%s", want, result.Manifest)
		}
	}
}

func TestPlanBuildsInternetRoute(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName:         "demo",
		CWD:                 dir,
		Exposure:            "internet",
		ImageTag:            "test",
		CloudflareToken:     true,
		CloudflareAccountID: "account-id",
		CloudflareTunnelID:  "tunnel-id",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Exposure != "internet" || result.TailscaleOnly {
		t.Fatalf("unexpected exposure %q tailscaleOnly=%t", result.Exposure, result.TailscaleOnly)
	}
	for _, want := range []string{
		`ship.local/exposure: "internet"`,
		`ship.local/tailscale-only: "false"`,
		"name: ship-tailscale",
	} {
		if !strings.Contains(result.Manifest, want) {
			t.Fatalf("manifest missing %q:\n%s", want, result.Manifest)
		}
	}
	for _, notWant := range []string{
		"kind: Ingress",
		`tailscale.com/funnel: "true"`,
		"ingressClassName: tailscale",
	} {
		if strings.Contains(result.Manifest, notWant) {
			t.Fatalf("internet manifest should not include %q:\n%s", notWant, result.Manifest)
		}
	}
	commands := strings.Join(result.Commands, "\n")
	if !strings.Contains(commands, "cloudflare tunnel expose demo.example.com") {
		t.Fatalf("internet route plan missing Cloudflare tunnel command:\n%s", commands)
	}
}

func TestPlanAllowsK8sInternetRouteWhenDashboardServiceIsConfiguredElsewhere(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName:      "k8s",
		CWD:              dir,
		Exposure:         "internet",
		DryRun:           true,
		DashboardService: "ops",
		ImageTag:         "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Exposure != "internet" {
		t.Fatalf("unexpected exposure %q", result.Exposure)
	}
}

func TestPlanUsesConfiguredDomainAndImagePrefix(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName: "demo",
		CWD:         dir,
		Domain:      "mydomain.com",
		ImagePrefix: "myorg",
		ImageTag:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Host != "demo.mydomain.com" {
		t.Fatalf("unexpected host %q", result.Host)
	}
	if result.Image != "myorg/demo:test" {
		t.Fatalf("unexpected image %q", result.Image)
	}
}

func TestPlanAllowsK8sDashboardServiceName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\nEXPOSE 3000\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName: "k8s",
		CWD:         dir,
		Domain:      "example.com",
		ImageTag:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Host != "k8s.example.com" {
		t.Fatalf("unexpected host %q", result.Host)
	}
}

func TestPlanMountsEnvFileAsSecretReference(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("SECRET_TOKEN=hidden\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName: "demo",
		CWD:         dir,
		EnvFile:     envFile,
		ImageTag:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.Manifest, "name: demo-env") {
		t.Fatalf("manifest missing secret reference:\n%s", result.Manifest)
	}
	if strings.Contains(result.Manifest, "hidden") {
		t.Fatalf("manifest leaked env value:\n%s", result.Manifest)
	}
	if result.EnvFilePath != envFile {
		t.Fatalf("unexpected env file path %q", result.EnvFilePath)
	}
}

func TestPlanUsesConfiguredServiceAccount(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{
		ServiceName:    "demo",
		CWD:            dir,
		ServiceAccount: "demo-reader",
		ImageTag:       "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.Manifest, "serviceAccountName: demo-reader") {
		t.Fatalf("manifest missing service account:\n%s", result.Manifest)
	}
}

func TestPlanInfersPortFromDockerfileExpose(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\nEXPOSE 3131\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Plan(Options{ServiceName: "demo", CWD: dir, ImageTag: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Port != 3131 {
		t.Fatalf("expected inferred port 3131, got %d", result.Port)
	}
	if !strings.Contains(result.Manifest, "containerPort: 3131") {
		t.Fatalf("manifest missing inferred port:\n%s", result.Manifest)
	}
}

func TestPlanRejectsInvalidServiceName(t *testing.T) {
	_, err := Plan(Options{ServiceName: "Bad_Name"})
	if err == nil {
		t.Fatal("expected invalid service name error")
	}
}

func TestPlanRejectsInvalidExposure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Plan(Options{ServiceName: "demo", CWD: dir, Exposure: "public"})
	if err == nil {
		t.Fatal("expected invalid exposure error")
	}
}
