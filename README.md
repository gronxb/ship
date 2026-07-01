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

Before you start, have:

- a domain managed by Cloudflare, such as `your-domain.com`
- a Tailscale account where you can create tags and OAuth credentials

Install the CLI on macOS or Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
```

Get the credentials, then create `.env` from the `.env.example` shape:

- Cloudflare: create an
  [API token](https://developers.cloudflare.com/fundamentals/api/get-started/create-token/)
  from the Edit zone DNS template. Ship needs Zone DNS Edit for your domain,
  and Zone Read too unless you set `CLOUDFLARE_ZONE_ID`.
- Tailscale: follow the
  [Kubernetes Operator credential setup](https://tailscale.com/docs/kubernetes-operator/install-operator#configure-tags-and-oauth-credentials):
  add `tag:k8s-operator` and `tag:k8s`, then create an OAuth client with write
  access for General/Services, Devices/Core, and Keys/Auth Keys on
  `tag:k8s-operator`.

```env
SHIP_DOMAIN=your-domain.com
CLOUDFLARE_API_TOKEN=your-cloudflare-token
TAILSCALE_CLIENT_ID=your-tailscale-client-id
TAILSCALE_CLIENT_SECRET=your-tailscale-client-secret

# Optional dashboard service name.
# Defaults to k8s, which gives you k8s.your-domain.com.
# SHIP_DASHBOARD_SERVICE=ops
```

Then run:

```sh
ship install
```

Ship can deploy any project that has a `Dockerfile`; the framework and runtime
are up to you. This example uses a Hono hello-world app on Bun. Hono is a small
web framework for the Web Platform, and its
[Bun guide](https://hono.dev/docs/getting-started/bun) starts from the same tiny
app shape.

```sh
bun create hono@latest demo
cd demo
bun install
```

Use the Bun template, then make sure `src/index.ts` returns a simple response:

```ts
import { Hono } from 'hono'

const app = new Hono()

app.get('/', (c) => c.text('Hello Ship!'))

export default app
```

Add a minimal `Dockerfile`:

```Dockerfile
FROM oven/bun:1

WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --production
COPY . .

EXPOSE 3000
CMD ["bun", "run", "src/index.ts"]
```

Deploy it:

```sh
ship --service demo
```

Open `https://demo.your-domain.com` to see `Hello Ship!`. For your own app, keep
the same pattern: add a `Dockerfile`, then run `ship --service <name>`.

## Agent Skill

Install the Ship skill once:

```sh
npx skills add gronxb/ship
```

Then open a project in Claude Code, Codex, or another agent that can use the
skill:

```text
$ship deploy this project as demo
```

That is enough. The skill can create a suitable `Dockerfile` when the project
does not have one, deploy with Ship, and give you
`https://demo.your-domain.com`.

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
