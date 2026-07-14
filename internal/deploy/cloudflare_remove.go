package deploy

import (
	"context"
	"fmt"
)

type RemoveTunnelRouteOptions struct {
	Host         string
	Domain       string
	APIToken     string
	ZoneID       string
	AccountID    string
	TunnelID     string
	RemoveTunnel bool
	APIEndpoint  string
}

func RemoveCloudflareRoute(ctx context.Context, opts RemoveTunnelRouteOptions) error {
	if opts.APIToken == "" {
		return fmt.Errorf("missing CLOUDFLARE_API_TOKEN with Zone DNS Edit")
	}
	if opts.Host == "" || opts.Domain == "" {
		return fmt.Errorf("host and domain are required for Cloudflare cleanup")
	}
	if opts.RemoveTunnel && (opts.AccountID == "" || opts.TunnelID == "") {
		return fmt.Errorf("missing Cloudflare account or tunnel ID for internet route cleanup")
	}
	endpoint := opts.APIEndpoint
	if endpoint == "" {
		endpoint = CloudflareAPIEndpoint
	}
	client := newCloudflareClient(endpoint, opts.APIToken)
	if opts.RemoveTunnel {
		current, err := client.tunnelConfiguration(ctx, opts.AccountID, opts.TunnelID)
		if err != nil {
			return err
		}
		config := cloudflareTunnelConfig{Ingress: removeTunnelIngress(current.Config.Ingress, opts.Host)}
		if err := client.putTunnelConfiguration(ctx, opts.AccountID, opts.TunnelID, config); err != nil {
			return err
		}
	}
	zoneID := opts.ZoneID
	if zoneID == "" {
		var err error
		zoneID, err = client.zoneID(ctx, opts.Domain)
		if err != nil {
			return err
		}
	}
	records, err := client.records(ctx, zoneID, opts.Host)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := client.deleteRecord(ctx, zoneID, record.ID); err != nil {
			return err
		}
	}
	return nil
}

func removeTunnelIngress(existing []cloudflareIngressRule, host string) []cloudflareIngressRule {
	rules := make([]cloudflareIngressRule, 0, len(existing)+1)
	for _, rule := range existing {
		if rule.Hostname == host || isTunnelCatchAll(rule) {
			continue
		}
		rules = append(rules, rule)
	}
	return append(rules, cloudflareIngressRule{Service: "http_status:404"})
}
