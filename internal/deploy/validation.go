package deploy

import (
	"fmt"
	"strings"
	"unicode"
)

func validateManifestOptions(opts Options) error {
	if err := validateDNSLabel("service name", opts.ServiceName); err != nil {
		return err
	}
	if err := validateDNSLabel("namespace", opts.Namespace); err != nil {
		return err
	}
	if err := validateDNSSubdomain("domain", opts.Domain); err != nil {
		return err
	}
	if len(opts.ServiceName)+1+len(opts.Domain) > 253 {
		return fmt.Errorf("hostname must be at most 253 characters")
	}
	for _, value := range []struct {
		field string
		value string
	}{
		{field: "gateway name", value: opts.GatewayName},
		{field: "internet gateway", value: opts.InternetGateway},
		{field: "service account", value: opts.ServiceAccount},
	} {
		if value.value == "" {
			continue
		}
		if err := validateDNSSubdomain(value.field, value.value); err != nil {
			return err
		}
	}
	if err := validateDNSLabel("gateway namespace", opts.GatewayNamespace); err != nil {
		return err
	}
	for _, value := range []struct {
		field string
		value string
	}{
		{field: "image prefix", value: opts.ImagePrefix},
		{field: "registry", value: opts.Registry},
		{field: "image tag", value: opts.ImageTag},
	} {
		if strings.IndexFunc(value.value, func(r rune) bool {
			return unicode.IsControl(r) || unicode.IsSpace(r)
		}) >= 0 {
			return fmt.Errorf("%s must not contain whitespace or control characters", value.field)
		}
	}
	return nil
}

func validateDNSLabel(field string, value string) error {
	if len(value) == 0 || len(value) > 63 || !serviceNamePattern.MatchString(value) {
		return fmt.Errorf("%s must be a DNS label of at most 63 lowercase letters, numbers, or hyphens", field)
	}
	return nil
}

func validateDNSSubdomain(field string, value string) error {
	if len(value) == 0 || len(value) > 253 {
		return fmt.Errorf("%s must be a DNS name of at most 253 characters", field)
	}
	for _, label := range strings.Split(value, ".") {
		if err := validateDNSLabel(field, label); err != nil {
			return err
		}
	}
	return nil
}
