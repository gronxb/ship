package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gronxb/ship/internal/deploy"
)

func runDNS(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "publish" {
		return fmt.Errorf("usage: ship dns publish --record <name> --target <address>")
	}

	flags := flag.NewFlagSet("ship dns publish", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	config := loadConfig()
	record := flags.String("record", "", "DNS record name to publish")
	target := flags.String("target", "", "DNS record target")
	domain := flags.String("domain", configDefault(config, "SHIP_DOMAIN", ""), "Cloudflare zone domain")
	zoneID := flags.String("zone-id", configDefault(config, "CLOUDFLARE_ZONE_ID", configDefault(config, "CF_ZONE_ID", "")), "Cloudflare zone id")
	apiEndpoint := flags.String("api-endpoint", deploy.CloudflareAPIEndpoint, "Cloudflare API endpoint")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if *record == "" {
		return fmt.Errorf("--record is required")
	}
	if *target == "" {
		return fmt.Errorf("--target is required")
	}
	if *domain == "" && *zoneID == "" {
		return fmt.Errorf("SHIP_DOMAIN or --zone-id is required")
	}

	return deploy.PublishCloudflareRecord(ctx, deploy.DNSRecordOptions{
		Domain:      *domain,
		RecordName:  *record,
		Target:      *target,
		APIToken:    cloudflareToken(config),
		ZoneID:      *zoneID,
		Output:      os.Stdout,
		APIEndpoint: *apiEndpoint,
	})
}
