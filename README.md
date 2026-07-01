# Ship

Ship turns any local project with a `Dockerfile` into a Kubernetes service at
`<service>.<your-domain>`. It is built for small self-hosted clusters, Mac mini
homelabs, and teams that want a thin deployment path without introducing a full
PaaS.

The project has three parts:

- `cmd/ship`: CLI that builds an image, applies Kubernetes resources, and records
  deployment state.
- `deploy-system`: Gateway, dashboard bootstrap, and DNS helper scripts.
- `start-app`: dashboard for deployed containers, exposure controls, network
  requests, terminal commands, manifests, and container logs.

## Status

Ship is production-oriented but intentionally small. The default exposure mode is
tailnet-only through the Tailscale Kubernetes Operator. If
`CLOUDFLARE_API_TOKEN` is set, onboarding creates the wildcard DNS record in
Cloudflare; otherwise it prints the exact record to create manually.

## Requirements

- Go 1.22 or newer
- Docker
- kubectl access to the target cluster
- kind or a container registry
- Helm for local cluster bootstrap
- Kubernetes Gateway API, Envoy Gateway, and Tailscale Kubernetes Operator for
  the default private Gateway
- cert-manager and a wildcard certificate issuer
- pnpm 10 or newer for dashboard development

## Quick Start

For a step-by-step installation guide, including an agent prompt,
troubleshooting, and uninstall notes, see
[docs/guide/installation.md](docs/guide/installation.md).

### For humans

Install the CLI:

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
```

Fill `.env`:

```sh
SHIP_DOMAIN=mydomain.com
CLOUDFLARE_API_TOKEN=<token>
TAILSCALE_CLIENT_ID=<client-id>
TAILSCALE_CLIENT_SECRET=<client-secret>
# Optional; defaults to k8s.
# SHIP_DASHBOARD_SERVICE=ops
```

Run 0 to 100:

```sh
ship install
```

Then verify:

```sh
ship --help
kubectl rollout status deployment/k8s -n ship-services --timeout=180s
open https://k8s.mydomain.com
```

Deploy any Dockerfile project after onboarding:

```sh
cd /path/to/app
ship --service demo
```

Ship creates a `Deployment`, `Service`, and `HTTPRoute` for
`https://demo.mydomain.com`.

### For agents

Use this prompt with an LLM/coding agent that has terminal access:

```text
Set up Ship from a blank local Kubernetes environment.
Ask me for my Cloudflare-managed base domain, Cloudflare API token, and Tailscale Kubernetes Operator OAuth client id/secret if I did not provide them.
Run:
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh
Then export PATH="$HOME/.local/bin:$PATH", create .env with SHIP_DOMAIN, CLOUDFLARE_API_TOKEN, TAILSCALE_CLIENT_ID, and TAILSCALE_CLIENT_SECRET, run ship install, verify deployment/k8s is rolled out, and open or curl https://k8s.<my-domain>.
Use ship uninstall for teardown.
```

## Configuration

Ship reads `~/.config/ship/config.env`, then lets environment variables and CLI
flags override those values.

```sh
SHIP_DOMAIN=mydomain.com
CLOUDFLARE_API_TOKEN=
TAILSCALE_CLIENT_ID=
TAILSCALE_CLIENT_SECRET=
# Optional; defaults to k8s.
# SHIP_DASHBOARD_SERVICE=ops
```

Common overrides:

```sh
ship --service demo --domain mydomain.com
ship --service demo --port 3000
ship --service demo --exposure internet
ship --service demo --env-file .env.production
ship --service demo --dry-run --json
```

## Dashboard

The dashboard shows deployed service cards, network request traces, terminal
commands, rendered manifests, recent container logs, and a control for promoting
a Tailscale-only route through Tailscale Funnel. It still does not deploy new
workloads from the browser. Onboarding deploys it as the normal Ship service
`k8s`, so `https://k8s.mydomain.com` is the first verification target.

Cluster names and service names are separate. A local kind cluster should be
named `ship`, which creates the Kubernetes context `kind-ship`; services inside
it can still be named `k8s`, `e2e`, `nitro-app`, or any other `ship --service`
value.

Redeploy it manually:

```sh
cd deploy-system
./deploy-dashboard.sh
```

Run it locally only for dashboard development:

```sh
cd start-app
pnpm install
pnpm dev --host 0.0.0.0 --port 3000
```

## Private And Public Exposure

### Tailscale Mode

Tailscale mode is the default:

- only devices in your tailnet can reach deployed services
- `*.mydomain.com` resolves to the Tailscale Gateway address
- Dockerfile projects do not need host port mapping

`deploy-domain.sh` applies the Gateway and prints the wildcard DNS record by
default. Create that record wherever the domain is managed:

```sh
cd deploy-system
./deploy-domain.sh
```

Deploy the dashboard after the Gateway:

```sh
./deploy-domain.sh
./deploy-dashboard.sh
```

### Internet Mode

Internet exposure setup is one command for everything Ship can automate:

```sh
cd deploy-system && ./deploy-internet-gateway.sh
```

The script verifies the Tailscale IngressClass and prints the remaining manual
tailnet policy action for Funnel. Ship's public route uses a Tailscale Funnel
Ingress, so local kind clusters do not need a public LoadBalancer.

Then opt a service into public routing:

```sh
ship --service demo --exposure internet
```

## Development

Run the full local verification suite:

```sh
make test
```

Run focused checks:

```sh
make go-test
make dashboard-test
make readiness
```

The release-readiness gate checks that the repository has the expected
open-source metadata, CI workflow, install path, dashboard scripts, and non-root
dashboard container configuration.

Bootstrap a blank local kind cluster with Ship's required Gateway stack:

```sh
TAILSCALE_CLIENT_ID=<client-id> TAILSCALE_CLIENT_SECRET=<client-secret> ./scripts/bootstrap-kind.sh
```

## Repository Layout

```text
cmd/ship/          CLI entrypoint
internal/deploy/   deployment planner and Kubernetes manifest renderer
deploy-system/     Gateway, dashboard bootstrap, DNS, and validation scripts
start-app/         dashboard
scripts/           repository maintenance checks
```

## Security

Ship shells out to Docker, kubectl, kind, and registry tooling from the machine
where it runs. Treat that host as part of your deployment trust boundary. Review
generated manifests with `ship --dry-run --json` before applying changes to a
new cluster.

Report vulnerabilities using the process in [SECURITY.md](SECURITY.md).

## Contributing

Issues and pull requests are welcome. Start with
[CONTRIBUTING.md](CONTRIBUTING.md) and run `make test` before submitting a PR.

## License

MIT. See [LICENSE](LICENSE).
