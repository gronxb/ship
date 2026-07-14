package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublishCloudflareRecordPatchesARecord(t *testing.T) {
	var patched bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("missing auth header: %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			w.Write([]byte(`{"success":true,"result":[{"id":"zone-id","name":"example.com"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/zones/zone-id/dns_records":
			w.Write([]byte(`{"success":true,"result":[{"id":"record-id","type":"A"}]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/zones/zone-id/dns_records/record-id":
			patched = true
			w.Write([]byte(`{"success":true,"result":{"id":"record-id"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	err := PublishCloudflareRecord(context.Background(), DNSRecordOptions{
		Domain:      "example.com",
		RecordName:  "demo.example.com",
		Target:      "100.124.154.47",
		APIToken:    "test-token",
		Output:      &output,
		APIEndpoint: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !patched {
		t.Fatal("expected existing record to be patched")
	}
	if !strings.Contains(output.String(), "updated: demo.example.com A 100.124.154.47 proxied=false") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestExposeCloudflareTunnelRoutePublishesSameHostname(t *testing.T) {
	var putConfig map[string]any
	var dnsBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			w.Write([]byte(`{"success":true,"result":{"config":{"ingress":[{"hostname":"old.example.com","service":"http://old.ship-services.svc.cluster.local:80"},{"service":"http_status:404"}]}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			if err := json.NewDecoder(r.Body).Decode(&putConfig); err != nil {
				t.Fatal(err)
			}
			w.Write([]byte(`{"success":true,"result":{"config":{"ingress":[]}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/zones/zone-id/dns_records":
			w.Write([]byte(`{"success":true,"result":[{"id":"old-a","type":"A"}]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/zones/zone-id/dns_records/old-a":
			w.Write([]byte(`{"success":true,"result":{"id":"old-a"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/zones/zone-id/dns_records":
			if err := json.NewDecoder(r.Body).Decode(&dnsBody); err != nil {
				t.Fatal(err)
			}
			w.Write([]byte(`{"success":true,"result":{"id":"record-id"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	err := ExposeCloudflareTunnelRoute(context.Background(), TunnelRouteOptions{
		Host:        "demo.example.com",
		ServiceName: "demo",
		Namespace:   "ship-services",
		APIToken:    "test-token",
		ZoneID:      "zone-id",
		AccountID:   "account-id",
		TunnelID:    "tunnel-id",
		Output:      &output,
		APIEndpoint: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	rawConfig, err := json.Marshal(putConfig)
	if err != nil {
		t.Fatal(err)
	}
	config := string(rawConfig)
	for _, want := range []string{
		`"hostname":"old.example.com"`,
		`"hostname":"demo.example.com"`,
		`"service":"http://demo.ship-services.svc.cluster.local:80"`,
		`"service":"http_status:404"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("tunnel config missing %q:\n%s", want, config)
		}
	}
	rawDNS, err := json.Marshal(dnsBody)
	if err != nil {
		t.Fatal(err)
	}
	dns := string(rawDNS)
	for _, want := range []string{
		`"type":"CNAME"`,
		`"name":"demo.example.com"`,
		`"content":"tunnel-id.cfargotunnel.com"`,
		`"proxied":true`,
	} {
		if !strings.Contains(dns, want) {
			t.Fatalf("dns record missing %q:\n%s", want, dns)
		}
	}
	if !strings.Contains(output.String(), "created: demo.example.com CNAME tunnel-id.cfargotunnel.com proxied=true") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestRemoveCloudflareRouteDeletesTunnelIngressAndDNSRecords(t *testing.T) {
	// Given a public Ship route and two DNS records for the same host.
	deleted := map[string]bool{}
	var putConfig map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			w.Write([]byte(`{"success":true,"result":{"config":{"ingress":[{"hostname":"demo.example.com","service":"http://demo.ship-services.svc.cluster.local:80"},{"hostname":"keep.example.com","service":"http://keep.ship-services.svc.cluster.local:80"},{"service":"http_status:404"}]}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			if err := json.NewDecoder(r.Body).Decode(&putConfig); err != nil {
				t.Fatal(err)
			}
			w.Write([]byte(`{"success":true,"result":{}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/zones/zone-id/dns_records":
			w.Write([]byte(`{"success":true,"result":[{"id":"record-a","type":"A"},{"id":"record-cname","type":"CNAME"}]}`))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/zones/zone-id/dns_records/"):
			deleted[strings.TrimPrefix(r.URL.Path, "/zones/zone-id/dns_records/")] = true
			w.Write([]byte(`{"success":true,"result":{}}`))
		default:
			t.Fatalf("unexpected Cloudflare request: %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	// When the service route is removed.
	err := RemoveCloudflareRoute(context.Background(), RemoveTunnelRouteOptions{
		Host:         "demo.example.com",
		Domain:       "example.com",
		APIToken:     "token",
		ZoneID:       "zone-id",
		AccountID:    "account-id",
		TunnelID:     "tunnel-id",
		RemoveTunnel: true,
		APIEndpoint:  server.URL,
	})

	// Then only that tunnel ingress disappears and every exact DNS record is deleted.
	if err != nil {
		t.Fatal(err)
	}
	rawConfig, err := json.Marshal(putConfig)
	if err != nil {
		t.Fatal(err)
	}
	config := string(rawConfig)
	if strings.Contains(config, "demo.example.com") || !strings.Contains(config, "keep.example.com") || !strings.Contains(config, "http_status:404") {
		t.Fatalf("unexpected tunnel cleanup body: %s", config)
	}
	for _, id := range []string{"record-a", "record-cname"} {
		if !deleted[id] {
			t.Fatalf("DNS record %s was not deleted: %v", id, deleted)
		}
	}
}

func TestPublishCloudflareRecordDeletesDuplicateSameTypeRecords(t *testing.T) {
	deleted := map[string]bool{}
	var patched bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			w.Write([]byte(`{"success":true,"result":[{"id":"zone-id","name":"example.com"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/zones/zone-id/dns_records":
			w.Write([]byte(`{"success":true,"result":[{"id":"keep-id","type":"A"},{"id":"stale-id","type":"A"},{"id":"cname-id","type":"CNAME"}]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/zones/zone-id/dns_records/keep-id":
			patched = true
			w.Write([]byte(`{"success":true,"result":{"id":"keep-id"}}`))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/zones/zone-id/dns_records/"):
			deleted[strings.TrimPrefix(r.URL.Path, "/zones/zone-id/dns_records/")] = true
			w.Write([]byte(`{"success":true,"result":{"id":"deleted"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	err := PublishCloudflareRecord(context.Background(), DNSRecordOptions{
		Domain:      "example.com",
		RecordName:  "demo.example.com",
		Target:      "100.124.154.47",
		APIToken:    "test-token",
		Output:      &output,
		APIEndpoint: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !patched {
		t.Fatal("expected canonical A record to be patched")
	}
	for _, id := range []string{"stale-id", "cname-id"} {
		if !deleted[id] {
			t.Fatalf("expected %s to be deleted; deleted=%v", id, deleted)
		}
	}
	if !strings.Contains(output.String(), "updated: demo.example.com A 100.124.154.47 proxied=false") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}
