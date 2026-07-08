package deploy

import "regexp"

type Options struct {
	ServiceName         string
	CWD                 string
	Port                int
	DryRun              bool
	JSON                bool
	Namespace           string
	Domain              string
	GatewayNamespace    string
	GatewayName         string
	InternetGateway     string
	Exposure            string
	ImagePrefix         string
	Registry            string
	ImageTag            string
	KindCluster         string
	EnvFile             string
	ServiceAccount      string
	DNSMode             string
	CloudflareToken     bool
	CloudflareAPIKey    string
	CloudflareZoneID    string
	CloudflareAccountID string
	CloudflareTunnelID  string
	ShipCommand         string
	DashboardService    string
}

type Result struct {
	ServiceName    string   `json:"serviceName"`
	Host           string   `json:"host"`
	Image          string   `json:"image"`
	Namespace      string   `json:"namespace"`
	DockerfilePath string   `json:"dockerfilePath"`
	ContextDir     string   `json:"contextDir"`
	EnvFilePath    string   `json:"envFilePath,omitempty"`
	Port           int      `json:"port"`
	Exposure       string   `json:"exposure"`
	TailscaleOnly  bool     `json:"tailscaleOnly"`
	DryRun         bool     `json:"dryRun"`
	Commands       []string `json:"commands"`
	Manifest       string   `json:"manifest"`
}

var serviceNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
