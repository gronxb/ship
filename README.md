# Ship

Ship turns any local project with a `Dockerfile` into a Kubernetes service at
`<service>.<your-domain>`. It is built for small self-hosted clusters, Mac mini
homelabs, and teams that want a thin deployment path without introducing a full
PaaS.

The project has three parts:

- `cmd/ship`: CLI that builds an image, applies Kubernetes resources, and records
  deployment state.
- `deploy-system`: Gateway, proxy, dashboard route, and DNS helper scripts.
- `start-app`: read-only dashboard for deployed containers, network requests,
  terminal commands, manifests, and container logs.

## Status

Ship is production-oriented but intentionally small. It assumes you already own
the cluster and DNS boundary. The default exposure mode is tailnet-only through
the Tailscale Kubernetes Operator; public internet exposure is opt-in and should
only be enabled on clusters with a real public LoadBalancer.

## Requirements

- Go 1.22 or newer
- Docker
- kubectl access to the target cluster
- kind or a container registry
- Kubernetes Gateway API
- Envoy Gateway
- Tailscale Kubernetes Operator for the default private Gateway
- cert-manager and a wildcard certificate issuer
- Wrangler login when using the Cloudflare DNS helper
- pnpm 10 or newer for dashboard development

## Quick Start

Install the CLI:

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | SHIP_DOMAIN=mydomain.com sh
export PATH="$HOME/.local/bin:$PATH"
```

Review the generated config:

```sh
$EDITOR ~/.config/ship/config.env
```

Deploy the Gateway once:

```sh
git clone https://github.com/gronxb/ship.git
cd ship/deploy-system
./deploy-domain.sh
```

Deploy any project that contains a `Dockerfile`:

```sh
cd /path/to/app
ship --service demo
```

Ship creates a `Deployment`, `Service`, and `HTTPRoute` for
`https://demo.mydomain.com`.

## Configuration

Ship reads `~/.config/ship/config.env`, then lets environment variables and CLI
flags override those values.

```sh
SHIP_DOMAIN=mydomain.com
SHIP_NAMESPACE=ship-services
SHIP_GATEWAY_NAMESPACE=ship-system
SHIP_GATEWAY_NAME=ship-tailscale
SHIP_INTERNET_GATEWAY_NAME=ship-internet
SHIP_DASHBOARD_HOST=k8s.mydomain.com
SHIP_DASHBOARD_PORT=9292
SHIP_IMAGE_PREFIX=ship
KIND_CLUSTER=ship
REGISTRY=
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

The dashboard is intentionally read-only. It shows deployed service cards,
network request traces, terminal commands, rendered manifests, and recent
container logs without deploying new workloads from the browser.

Run it locally:

```sh
cd start-app
pnpm install
pnpm dev --host 0.0.0.0 --port 9292
```

After `deploy-system` is applied, open `https://k8s.mydomain.com` from your
tailnet.

## Private And Public Exposure

### Tailscale Mode

Tailscale mode is the default:

- only devices in your tailnet can reach deployed services
- `*.mydomain.com` resolves to the Tailscale Gateway address
- Dockerfile projects do not need host port mapping

The deploy script publishes a DNS-only Cloudflare wildcard record when Wrangler
is logged in:

```sh
cd deploy-system
./cloudflare-login.sh
./deploy-domain.sh
```

### Internet Mode

Internet exposure requires a real public LoadBalancer:

```sh
cd deploy-system
./render-internet-gateway.sh | kubectl apply -f -
kubectl get gateway -n ship-system ship-internet
```

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

## Repository Layout

```text
cmd/ship/          CLI entrypoint
internal/deploy/   deployment planner and Kubernetes manifest renderer
deploy-system/     Gateway, proxy, DNS, and validation scripts
start-app/         read-only dashboard
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
