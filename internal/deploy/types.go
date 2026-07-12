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
	ComposeFile         string
	ComposeService      string
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
	ServiceName     string   `json:"serviceName"`
	Runtime         string   `json:"runtime"`
	Host            string   `json:"host"`
	Image           string   `json:"image,omitempty"`
	Namespace       string   `json:"namespace"`
	DockerfilePath  string   `json:"dockerfilePath,omitempty"`
	ComposeFilePath string   `json:"composeFilePath,omitempty"`
	ComposeService  string   `json:"composeService,omitempty"`
	PublishedPort   int      `json:"publishedPort,omitempty"`
	HostGateway     string   `json:"hostGateway,omitempty"`
	ContextDir      string   `json:"contextDir"`
	EnvFilePath     string   `json:"envFilePath,omitempty"`
	Port            int      `json:"port"`
	Exposure        string   `json:"exposure"`
	TailscaleOnly   bool     `json:"tailscaleOnly"`
	DryRun          bool     `json:"dryRun"`
	Commands        []string `json:"commands"`
	Manifest        string   `json:"manifest"`
}

var serviceNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
