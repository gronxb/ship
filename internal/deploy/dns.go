package deploy

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"strings"
)

const CloudflareAPIEndpoint = "https://api.cloudflare.com/client/v4"

var (
	gatewayAddress   = kubectlGatewayAddress
	publishDNSRecord = PublishCloudflareRecord
)

type DNSRecordOptions struct {
	Domain      string
	RecordName  string
	Target      string
	APIToken    string
	ZoneID      string
	Proxied     bool
	TTL         int
	Comment     string
	Output      io.Writer
	APIEndpoint string
}

func kubectlGatewayAddress(ctx context.Context, namespace string, name string) (string, error) {
	output, err := commandOutput(ctx, "", "kubectl", "get", "gateway", name, "-n", namespace, "-o", "jsonpath={.status.addresses[0].value}")
	if err != nil {
		return "", err
	}
	address := strings.TrimSpace(output)
	if address == "" {
		return "", fmt.Errorf("gateway %s/%s has no published address", namespace, name)
	}
	return address, nil
}

func PublishCloudflareRecord(ctx context.Context, opts DNSRecordOptions) error {
	if opts.APIToken == "" {
		return fmt.Errorf("missing CLOUDFLARE_API_TOKEN with Zone DNS Edit")
	}
	if opts.RecordName == "" {
		return fmt.Errorf("record name is required")
	}
	if opts.Target == "" {
		return fmt.Errorf("record target is required")
	}
	endpoint := opts.APIEndpoint
	if endpoint == "" {
		endpoint = CloudflareAPIEndpoint
	}
	client := newCloudflareClient(endpoint, opts.APIToken)

	zoneID := opts.ZoneID
	if zoneID == "" {
		if opts.Domain == "" {
			return fmt.Errorf("SHIP_DOMAIN or CLOUDFLARE_ZONE_ID is required")
		}
		var err error
		zoneID, err = client.zoneID(ctx, opts.Domain)
		if err != nil {
			return err
		}
	}

	recordType, target := recordTypeAndTarget(opts.Target)
	records, err := client.records(ctx, zoneID, opts.RecordName)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Type == recordType {
			continue
		}
		if err := client.deleteRecord(ctx, zoneID, record.ID); err != nil {
			return err
		}
	}

	body := map[string]any{
		"type":    recordType,
		"name":    opts.RecordName,
		"content": target,
		"ttl":     dnsTTL(opts),
		"proxied": opts.Proxied,
		"comment": dnsComment(opts),
	}
	action := "created"
	updated := false
	for _, record := range records {
		if record.Type != recordType {
			continue
		}
		if updated {
			if err := client.deleteRecord(ctx, zoneID, record.ID); err != nil {
				return err
			}
			continue
		}
		if err := client.patchRecord(ctx, zoneID, record.ID, body); err != nil {
			return err
		}
		updated = true
		action = "updated"
	}
	if updated {
		return printDNSResult(opts.Output, action, opts.RecordName, recordType, target, opts.Proxied)
	}
	if err := client.createRecord(ctx, zoneID, body); err != nil {
		return err
	}
	return printDNSResult(opts.Output, action, opts.RecordName, recordType, target, opts.Proxied)
}

func recordTypeAndTarget(target string) (string, string) {
	target = strings.TrimSpace(target)
	if addr, err := netip.ParseAddr(target); err == nil {
		if addr.Is6() {
			return "AAAA", target
		}
		return "A", target
	}
	return "CNAME", strings.TrimSuffix(target, ".")
}

func printDNSResult(output io.Writer, action string, name string, recordType string, target string, proxied bool) error {
	if output == nil {
		output = os.Stdout
	}
	_, err := fmt.Fprintf(output, "%s: %s %s %s proxied=%t\n", action, name, recordType, target, proxied)
	return err
}

func dnsTTL(opts DNSRecordOptions) int {
	if opts.TTL > 0 {
		return opts.TTL
	}
	return 60
}

func dnsComment(opts DNSRecordOptions) string {
	if opts.Comment != "" {
		return opts.Comment
	}
	if opts.Proxied {
		return "Ship Cloudflare Tunnel public route."
	}
	return "Ship Kubernetes Gateway. DNS-only; do not proxy."
}

func commandOutput(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return string(output), nil
}
