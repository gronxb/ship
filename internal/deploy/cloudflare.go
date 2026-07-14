package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type cloudflareClient struct {
	endpoint string
	token    string
	client   *http.Client
}

type cloudflareRecord struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type TunnelRouteOptions struct {
	Host        string
	ServiceName string
	Namespace   string
	Domain      string
	APIToken    string
	ZoneID      string
	AccountID   string
	TunnelID    string
	Output      io.Writer
	APIEndpoint string
}

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

type cloudflareTunnelConfiguration struct {
	Config cloudflareTunnelConfig `json:"config"`
}

type cloudflareTunnelConfig struct {
	Ingress []cloudflareIngressRule `json:"ingress"`
}

type cloudflareIngressRule struct {
	Hostname string `json:"hostname,omitempty"`
	Service  string `json:"service"`
}

func newCloudflareClient(endpoint string, token string) *cloudflareClient {
	return &cloudflareClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		token:    token,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *cloudflareClient) zoneID(ctx context.Context, domain string) (string, error) {
	var response struct {
		Result []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	if err := c.request(ctx, http.MethodGet, "/zones?name="+url.QueryEscape(domain), nil, &response); err != nil {
		return "", err
	}
	for _, zone := range response.Result {
		if zone.Name == domain {
			return zone.ID, nil
		}
	}
	return "", fmt.Errorf("cloudflare zone not found: %s", domain)
}

func (c *cloudflareClient) records(ctx context.Context, zoneID string, name string) ([]cloudflareRecord, error) {
	var response struct {
		Result []cloudflareRecord `json:"result"`
	}
	path := "/zones/" + url.PathEscape(zoneID) + "/dns_records?name=" + url.QueryEscape(name) + "&per_page=100"
	if err := c.request(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (c *cloudflareClient) deleteRecord(ctx context.Context, zoneID string, recordID string) error {
	return c.request(ctx, http.MethodDelete, "/zones/"+url.PathEscape(zoneID)+"/dns_records/"+url.PathEscape(recordID), nil, nil)
}

func (c *cloudflareClient) patchRecord(ctx context.Context, zoneID string, recordID string, body map[string]any) error {
	return c.request(ctx, http.MethodPatch, "/zones/"+url.PathEscape(zoneID)+"/dns_records/"+url.PathEscape(recordID), body, nil)
}

func (c *cloudflareClient) createRecord(ctx context.Context, zoneID string, body map[string]any) error {
	return c.request(ctx, http.MethodPost, "/zones/"+url.PathEscape(zoneID)+"/dns_records", body, nil)
}

func (c *cloudflareClient) tunnelConfiguration(ctx context.Context, accountID string, tunnelID string) (cloudflareTunnelConfiguration, error) {
	var response struct {
		Result cloudflareTunnelConfiguration `json:"result"`
	}
	path := "/accounts/" + url.PathEscape(accountID) + "/cfd_tunnel/" + url.PathEscape(tunnelID) + "/configurations"
	if err := c.request(ctx, http.MethodGet, path, nil, &response); err != nil {
		return cloudflareTunnelConfiguration{}, err
	}
	return response.Result, nil
}

func (c *cloudflareClient) putTunnelConfiguration(ctx context.Context, accountID string, tunnelID string, config cloudflareTunnelConfig) error {
	body := cloudflareTunnelConfiguration{Config: config}
	path := "/accounts/" + url.PathEscape(accountID) + "/cfd_tunnel/" + url.PathEscape(tunnelID) + "/configurations"
	return c.request(ctx, http.MethodPut, path, body, nil)
}

func ExposeCloudflareTunnelRoute(ctx context.Context, opts TunnelRouteOptions) error {
	if opts.APIToken == "" {
		return fmt.Errorf("missing CLOUDFLARE_API_TOKEN with Zone DNS Edit and Cloudflare Tunnel Edit")
	}
	if opts.AccountID == "" {
		return fmt.Errorf("missing CLOUDFLARE_ACCOUNT_ID for Cloudflare Tunnel")
	}
	if opts.TunnelID == "" {
		return fmt.Errorf("missing CLOUDFLARE_TUNNEL_ID; run ship install")
	}
	if opts.Host == "" || opts.ServiceName == "" || opts.Namespace == "" {
		return fmt.Errorf("host, service name, and namespace are required for Cloudflare Tunnel exposure")
	}
	endpoint := opts.APIEndpoint
	if endpoint == "" {
		endpoint = CloudflareAPIEndpoint
	}
	client := newCloudflareClient(endpoint, opts.APIToken)
	current, err := client.tunnelConfiguration(ctx, opts.AccountID, opts.TunnelID)
	if err != nil {
		return err
	}
	config := cloudflareTunnelConfig{
		Ingress: mergeTunnelIngress(current.Config.Ingress, cloudflareIngressRule{
			Hostname: opts.Host,
			Service:  fmt.Sprintf("http://%s.%s.svc.cluster.local:80", opts.ServiceName, opts.Namespace),
		}),
	}
	if err := client.putTunnelConfiguration(ctx, opts.AccountID, opts.TunnelID, config); err != nil {
		return err
	}
	return PublishCloudflareRecord(ctx, DNSRecordOptions{
		Domain:      opts.Domain,
		RecordName:  opts.Host,
		Target:      opts.TunnelID + ".cfargotunnel.com",
		APIToken:    opts.APIToken,
		ZoneID:      opts.ZoneID,
		Proxied:     true,
		TTL:         1,
		Comment:     "Ship Cloudflare Tunnel public route.",
		Output:      opts.Output,
		APIEndpoint: endpoint,
	})
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

func mergeTunnelIngress(existing []cloudflareIngressRule, route cloudflareIngressRule) []cloudflareIngressRule {
	rules := make([]cloudflareIngressRule, 0, len(existing)+2)
	for _, rule := range existing {
		if rule.Hostname == route.Hostname || isTunnelCatchAll(rule) {
			continue
		}
		rules = append(rules, rule)
	}
	rules = append(rules, route, cloudflareIngressRule{Service: "http_status:404"})
	return rules
}

func isTunnelCatchAll(rule cloudflareIngressRule) bool {
	return rule.Hostname == "" && strings.HasPrefix(rule.Service, "http_status:")
}

func (c *cloudflareClient) request(ctx context.Context, method string, path string, body any, target any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("cloudflare api request: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var envelope struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("parse cloudflare response: %w", err)
	}
	if !envelope.Success {
		messages := make([]string, 0, len(envelope.Errors))
		for _, apiErr := range envelope.Errors {
			if apiErr.Message != "" {
				messages = append(messages, apiErr.Message)
			}
		}
		if len(messages) == 0 {
			messages = append(messages, string(raw))
		}
		hint := ""
		if res.StatusCode == http.StatusForbidden {
			hint = " Ensure CLOUDFLARE_API_TOKEN has Zone DNS Edit. If CLOUDFLARE_ZONE_ID is unset, add Zone Read or set CLOUDFLARE_ZONE_ID."
		}
		return fmt.Errorf("cloudflare api failed: %s%s", strings.Join(messages, "; "), hint)
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("parse cloudflare result: %w", err)
	}
	return nil
}
