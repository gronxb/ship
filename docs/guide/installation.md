# Installation

Ship installs as a small CLI first. After you fill `.env`, `ship install`
creates the local kind cluster, Gateway resources, Cloudflare wildcard DNS, and
dashboard at `https://k8s.<your-domain>`.

| You want | Run | Lands on disk or cluster |
| --- | --- | --- |
| CLI only on macOS/Linux | `curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh \| sh` | `~/.local/bin/ship` |
| CLI only on Windows | `go install github.com/gronxb/ship/cmd/ship@latest` | `%USERPROFILE%\go\bin\ship.exe` |
| 0 to 100 setup | `ship install` | `kind-ship`, Gateway resources, DNS, and dashboard |
| 100 to 0 teardown | `ship uninstall` | Removes DNS, cluster resources, and Ship config |
| Source checkout | `make test` then `go build -o ~/.local/bin/ship ./cmd/ship` | Local development binary |

## For Humans

Install the CLI on macOS or Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
ship -v
```

Install the CLI on Windows:

```powershell
go install github.com/gronxb/ship/cmd/ship@latest
$env:Path += ";$env:USERPROFILE\go\bin"
```

`ship install` bootstraps the kind cluster, Gateway stack, and dashboard through
POSIX shell scripts. Run it on macOS or Linux. On Windows, use the CLI against an
existing Kubernetes context from a Dockerfile project:

```powershell
ship --service demo
```

Fill `.env`:

```sh
SHIP_DOMAIN=mydomain.com
CLOUDFLARE_API_TOKEN=<token>
TAILSCALE_CLIENT_ID=<client-id>
TAILSCALE_CLIENT_SECRET=<client-secret>
# Optional dashboard service name.
# Defaults to k8s, which gives you k8s.your-domain.com.
# SHIP_DASHBOARD_SERVICE=ops
```

Run setup:

```sh
ship install
```

Then verify:

```sh
ship --help
kubectl rollout status deployment/k8s -n ship-services --timeout=180s
curl -fsS https://k8s.mydomain.com
```

Deploy any Dockerfile project after onboarding:

```sh
cd /path/to/app
ship --service demo
```

Ship creates a `Deployment`, `Service`, and `HTTPRoute` for
`https://demo.mydomain.com`.

For a Docker Compose project, run the same command from a directory with a
canonical Compose file, or select the source and routed service explicitly:

```sh
ship --service demo --compose-file ./docker-compose.yml --compose-service gateway --env-file ./.env
```

Ship keeps Compose on the local host and creates a selectorless `Service`, an
`EndpointSlice`, and an `HTTPRoute`. The selected service must publish a TCP
port on a non-loopback host address. Compose deployment currently requires the
active Kubernetes context to be the configured local kind cluster. Ship leaves
the Compose project name and persistent volumes under Compose ownership.

## For LLM Agents

Ship is designed so an LLM agent can complete the install end to end once the
user provides the target domain and permission to either create a local kind
cluster or use a named existing cluster. Fetch and read this guide, then follow
the steps. Do not summarize the install path; run it and verify it.

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/docs/guide/installation.md
```

Use this prompt when handing setup to an agent:

```text
Install and verify Ship. If no Kubernetes cluster exists, create a local kind
cluster named ship first.

1. Ask me for the Cloudflare-managed base domain if I did not provide one.
2. Ask me for Cloudflare API token and Tailscale Kubernetes Operator OAuth client id/secret if I did not provide them.
3. Check that go, docker, kubectl, kind, and helm are available for a local install.
4. Run `curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh`.
5. Export PATH="$HOME/.local/bin:$PATH".
6. Create `.env` with SHIP_DOMAIN, CLOUDFLARE_API_TOKEN, TAILSCALE_CLIENT_ID, and TAILSCALE_CLIENT_SECRET.
7. Run `ship install`.
8. Verify `kubectl rollout status deployment/k8s -n ship-services --timeout=180s`.
9. Open or curl https://k8s.<my-domain>.
10. Use `ship uninstall` for teardown.
```

### Agent runbook

1. Confirm the target domain.
   - If the user already gave a domain, use it.
   - If not, ask exactly once for the base domain, then continue.
2. Pick the cluster path:
   - For a new local install, create/select `kind-ship`.
   - For an existing remote cluster, use the context the user explicitly named
     or confirmed.
3. Check prerequisites:

   ```sh
   go version
   docker --version
   kubectl version --client
   helm version
   command -v kind || true
   ```

   If a required local binary is missing, install it only when the environment
   has an obvious package manager and the user already allows local installs.
   Otherwise report the missing binary and stop.
4. Install the CLI:

   ```sh
   curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh
   export PATH="$HOME/.local/bin:$PATH"
   ```

5. Create `.env`:

   ```sh
   SHIP_DOMAIN=<my-domain>
   CLOUDFLARE_API_TOKEN=<token>
   TAILSCALE_CLIENT_ID=<client-id>
   TAILSCALE_CLIENT_SECRET=<client-secret>
   ```

6. Run setup:

   ```sh
   ship install
   ```

   This creates `kind-ship` when needed, installs Envoy Gateway and the
   Tailscale Kubernetes Operator, applies Ship Gateway resources, creates
   Cloudflare DNS, and deploys the dashboard.
9. Verify the CLI and cluster rollout:

   ```sh
   ship --help
   kubectl config current-context
   kubectl get gateway -n ship-system
   kubectl rollout status deployment/k8s -n ship-services --timeout=180s
   ```

   For local kind installs, `kubectl config current-context` should be
   `kind-ship`.
10. Verify the user-facing dashboard:

   ```sh
   curl -I https://k8s.<my-domain>
   ```

   If `curl` fails immediately after DNS creation, retry after DNS propagation.
   If it still fails, inspect the Gateway address, wildcard DNS record, and
   dashboard rollout before changing anything.
11. Finish by telling the user:
   - where `ship` was installed
   - which Kubernetes context was used
   - the dashboard URL
   - the exact DNS record they created or still need to create
   - any failed check and the command output that proves it

## Prerequisites

- Go 1.22 or newer
- Docker
- `kubectl` access to the target cluster
- `kind` and Helm for local clusters, or a registry for non-kind clusters
- Kubernetes Gateway API
- Envoy Gateway
- Tailscale Kubernetes Operator for the default private Gateway
- Cloudflare DNS credentials for DNS automation and Let's Encrypt DNS-01
  wildcard Gateway TLS

The installer expects the current `kubectl` context to be the cluster you want
Ship to use. For local kind, use a cluster named `ship`, which creates the
context `kind-ship`. Deployed service names are separate from the cluster name:
`k8s`, `e2e`, and `nitro-app` are services inside the cluster, not cluster
names.

## Bootstrap A Blank Local Cluster

Use the repository checkout when starting from no Kubernetes cluster:

```sh
TAILSCALE_CLIENT_ID=<client-id> TAILSCALE_CLIENT_SECRET=<client-secret> ./scripts/bootstrap-kind.sh
```

The script:

- creates or reuses the `ship` kind cluster
- selects the `kind-ship` context
- installs Envoy Gateway with its Gateway API CRDs through the official Helm
  chart
- installs the Tailscale Kubernetes Operator through the official Helm chart
- verifies the CRDs and operator namespace before returning

Tailscale OAuth credentials are required because the operator must join your
tailnet. If they are missing, the script stops with the exact environment
variables to set and can be rerun safely.

## What The Installer Does

`install.sh` resolves the latest GitHub Release, downloads the matching
`ship_<version>_<os>_<arch>.tar.gz` binary asset, and installs it at
`~/.local/bin/ship`. This path does not require cloning the repository or
having Go installed. Set `SHIP_REF=main` only when you intentionally want a
development source build.

`ship install` reads `.env`, writes `~/.config/ship/config.env`, and runs:

```sh
scripts/bootstrap-kind.sh
deploy-system/deploy-domain.sh
deploy-system/deploy-cloudflare-tunnel.sh
deploy-system/deploy-dashboard.sh
```

That installs cert-manager with Gateway API support, creates a Let's Encrypt
Cloudflare DNS-01 issuer, configures public recursive resolvers for DNS-01
self-checks, applies the default Tailscale Gateway, waits for the wildcard
certificate used by `k8s.$SHIP_DOMAIN` and later `ship --service` routes,
creates Cloudflare DNS when `CLOUDFLARE_API_TOKEN` is set, creates or reuses a
Cloudflare Tunnel connector for explicit internet exposure, and deploys the
dashboard as the normal Ship service named `k8s`.

The values you normally fill are:

```sh
SHIP_DOMAIN=mydomain.com
CLOUDFLARE_API_TOKEN=
TAILSCALE_CLIENT_ID=
TAILSCALE_CLIENT_SECRET=
# Optional. ship install can usually infer these from the token and domain.
# CLOUDFLARE_ACCOUNT_ID=
# CLOUDFLARE_ZONE_ID=
# Optional; defaults to admin@$SHIP_DOMAIN.
# SHIP_ACME_EMAIL=admin@mydomain.com
# Optional dashboard service name.
# Defaults to k8s, which gives you k8s.your-domain.com.
# SHIP_DASHBOARD_SERVICE=ops
```

Environment variables and CLI flags override values from
`~/.config/ship/config.env`.

## Verify

Run the smallest checks first:

```sh
ship --help
kubectl get gateway -n ship-system
kubectl rollout status deployment/k8s -n ship-services --timeout=180s
```

Then check the real surface:

```sh
curl -I https://k8s.mydomain.com
```

For a dry-run deployment from any Dockerfile or Compose project:

```sh
ship --service demo --dry-run --json
```

## Public Internet Exposure

Ship defaults to Tailscale-only routing. `ship install` prepares Cloudflare
Tunnel automatically, but a service stays private until you expose that service:

```sh
ship --service demo --exposure internet
```

The hostname stays the same. By default, `demo.mydomain.com` reaches the
Tailscale Gateway through the wildcard DNS-only record. Public exposure adds a
specific proxied Cloudflare CNAME for `demo.mydomain.com` and routes it through
the Cloudflare Tunnel. Switching back removes that specific route so the
wildcard Tailscale path works again.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `ship: command not found` | Run `export PATH="$HOME/.local/bin:$PATH"` or add that line to your shell profile. |
| Installer prints `usage:` | Set `SHIP_DOMAIN=mydomain.com` or pass the domain with `sh -s -- mydomain.com`. |
| Dashboard does not load | Confirm the printed wildcard DNS record exists and points to the Gateway address. |
| Gateway has no address | Check the Tailscale Kubernetes Operator and Envoy Gateway installation for private routing; public exposure uses Cloudflare Tunnel after `ship install`. |
| Current context is `kind-e2e` | Create/select the local Ship cluster with `kind create cluster --name ship` and `kubectl config use-context kind-ship`; keep `e2e` for service names only. |
| Non-kind cluster cannot pull the image | Set `REGISTRY` or `SHIP_IMAGE_PREFIX` so the cluster can pull the built image. |
| Want to preview manifests | Run `ship --service demo --dry-run --json` before applying. |

## Maintenance

Update the CLI by rerunning the installer:

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | SHIP_DOMAIN=mydomain.com sh
```

Run repository checks from a source checkout:

```sh
make test
```

Focused checks:

```sh
make go-test
make dashboard-test
make readiness
make onboarding-smoke
```

## Uninstall

Remove the Ship-managed system:

```sh
ship uninstall
```

Remove only the local CLI binary:

```sh
rm -f ~/.local/bin/ship
```

`ship uninstall --dry-run` prints the DNS, kind, and config cleanup plan without
deleting anything.
