package deploy

import (
	"strings"
	"testing"
)

func TestValidateRejectsManifestStructuralInjection(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		mutate func(*Options)
	}{
		{name: "namespace", field: "namespace", mutate: func(opts *Options) { opts.Namespace = "ship-services\n---\nkind: Secret" }},
		{name: "domain", field: "domain", mutate: func(opts *Options) { opts.Domain = "example.com\n  rules: []" }},
		{name: "gateway name", field: "gateway name", mutate: func(opts *Options) { opts.GatewayName = "ship-tailscale\n---\nkind: Secret" }},
		{name: "gateway namespace", field: "gateway namespace", mutate: func(opts *Options) { opts.GatewayNamespace = "ship-system\n---\nkind: Secret" }},
		{name: "internet gateway", field: "internet gateway", mutate: func(opts *Options) { opts.InternetGateway = "ship-internet\n---\nkind: Secret" }},
		{name: "service account", field: "service account", mutate: func(opts *Options) { opts.ServiceAccount = "runner\n---\nkind: Secret" }},
		{name: "image prefix", field: "image prefix", mutate: func(opts *Options) { opts.ImagePrefix = "ship\nsecurityContext: {}" }},
		{name: "registry", field: "registry", mutate: func(opts *Options) { opts.Registry = "registry.example\nsecurityContext: {}" }},
		{name: "image tag", field: "image tag", mutate: func(opts *Options) { opts.ImageTag = "latest\nsecurityContext: {}" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := withDefaults(Options{ServiceName: "demo", DryRun: true, ImageTag: "test"})
			test.mutate(&opts)
			err := validate(opts)
			if err == nil || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("expected %s validation error, got %v", test.field, err)
			}
		})
	}
}

func TestValidateAcceptsDeploymentIdentifiersAndDomain(t *testing.T) {
	opts := withDefaults(Options{
		ServiceName:      "paca",
		Namespace:        "ship-services",
		Domain:           "gron-studio.com",
		GatewayName:      "ship-tailscale",
		GatewayNamespace: "ship-system",
		InternetGateway:  "ship-internet",
		ServiceAccount:   "paca-runner",
		Registry:         "registry.example.com/team",
		ImagePrefix:      "ship/apps",
		ImageTag:         "v1.2.3-build_4",
		DryRun:           true,
	})
	if err := validate(opts); err != nil {
		t.Fatalf("expected valid deployment options, got %v", err)
	}
}
