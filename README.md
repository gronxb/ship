# Ship

Ship turns any local project with a `Dockerfile` into a Kubernetes service under
`*.your-domain.com`. It is built for small self-hosted clusters, Mac mini
homelabs, and teams that want a thin deployment path without introducing a full
PaaS.

## Why Use Ship?

Ship is for deploying many apps from a self-hosted Mac mini or similar small
server. Services stay private inside your tailnet by default, and you can expose
them publicly only when you choose to.

## What You Get

- Dockerfile projects deployed as Kubernetes `Deployment`, `Service`, and
  `HTTPRoute` resources.
- Private-by-default routing through the Tailscale Kubernetes Operator.
- Optional public exposure through Tailscale Funnel.
- Deploys a dashboard for deployed services, exposure controls, requests,
  terminal commands, manifests, and logs.

## Quick Start

For prerequisites, `.env` values, Windows notes, troubleshooting, and uninstall
steps, see
[docs/guide/installation.md](docs/guide/installation.md).

Install the CLI on macOS or Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
```

Start from `.env.example`, then fill in your domain, Cloudflare token, and
Tailscale OAuth credentials:

```sh
cp .env.example .env
```

Then run:

```sh
ship install
```

Deploy any Dockerfile project:

```sh
cd /path/to/app
ship --service demo
```

Ship creates `https://demo.your-domain.com`.

## Exposure Model

Ship is private by default:

- `*.your-domain.com` resolves to the Tailscale Gateway address.
- only devices in your tailnet can reach deployed services.
- Dockerfile projects do not need host port mapping.

To expose one service publicly:

```sh
ship --service demo --exposure internet
```

## Development

Run tests:

```sh
make test
```

Main directories: `cmd/ship`, `internal/deploy`, `deploy-system`, `start-app`,
and `scripts`.

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
