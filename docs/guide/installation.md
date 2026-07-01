# Installation

Ship installs as a small CLI plus Kubernetes Gateway/dashboard resources. The
default path is tailnet-only: Ship prints the wildcard DNS record you need to
create, then deploys the dashboard at `https://k8s.<your-domain>`.

| You want | Run | Lands on disk or cluster |
| --- | --- | --- |
| CLI only | Installer with a domain argument | `~/.local/bin/ship` and `~/.config/ship/config.env` |
| Blank local kind cluster | `./scripts/bootstrap-kind.sh` | `kind-ship` plus Gateway API, Envoy Gateway, and Tailscale Operator |
| CLI + Gateway + dashboard | Installer with `SHIP_ONBOARD=1` | CLI/config plus Ship Gateway resources and the `k8s` dashboard service |
| Source checkout | `make test` then `go build -o ~/.local/bin/ship ./cmd/ship` | Local development binary |

## For Humans

Recommended one-liner:

For local kind installs with a Cloudflare-managed domain, fill the environment
values and run one command:

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | \
  CLOUDFLARE_API_TOKEN=<token> \
  SHIP_DOMAIN=mydomain.com \
  TS_OAUTH_CLIENT_ID=<client-id> \
  TS_OAUTH_CLIENT_SECRET=<client-secret> \
  SHIP_ONBOARD=1 \
  sh
```

The default dashboard service is `k8s`; set `SHIP_DASHBOARD_SERVICE=ops` to use
another name. Without Cloudflare, omit `CLOUDFLARE_API_TOKEN` and create the
printed wildcard DNS record manually.

```sh
curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | SHIP_DOMAIN=mydomain.com SHIP_ONBOARD=1 sh
export PATH="$HOME/.local/bin:$PATH"
```

Replace `mydomain.com` with the domain you want Ship to use. The installer
prints the manual wildcard DNS record:

```text
manual dns: create *.mydomain.com as DNS-only CNAME/A record to <gateway-address>
```

Create that record at your DNS provider, then verify:

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
4. Run:
   curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | CLOUDFLARE_API_TOKEN=<token> SHIP_DOMAIN=<my-domain> TS_OAUTH_CLIENT_ID=<client-id> TS_OAUTH_CLIENT_SECRET=<client-secret> SHIP_ONBOARD=1 sh
6. Export PATH="$HOME/.local/bin:$PATH".
7. Verify ship --help.
8. Verify kubectl rollout status deployment/k8s -n ship-services --timeout=180s.
9. If no Cloudflare token is provided and the installer prints "manual dns", stop and tell me the exact wildcard DNS record to create.
10. After DNS exists, open or curl https://k8s.<my-domain>.

Do not use browser-based deployment. The dashboard is deployed by the installer through ship.
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
4. For a new local kind install, bootstrap the cluster:

   ```sh
   TS_OAUTH_CLIENT_ID=<client-id> TS_OAUTH_CLIENT_SECRET=<client-secret> ./scripts/bootstrap-kind.sh
   kubectl get namespace
   ```

   The script creates `kind-ship` when needed, installs Envoy Gateway, and
   installs the Tailscale Kubernetes Operator when OAuth credentials are
   provided. Do not use service names such as `e2e`, `k8s`, or `nitro-app` as
   kind cluster names; those are deployed service names inside the cluster.
5. For an existing remote cluster, confirm cluster access:

   ```sh
   kubectl config current-context
   kubectl get namespace
   ```

   If this fails because the user is not logged in or the context is wrong,
   stop and ask for the correct Kubernetes context. This confirmation is only
   needed for existing or remote clusters, not for a new local kind cluster the
   agent just created.
6. Confirm the required cluster add-ons exist before running Ship onboarding:

   ```sh
   kubectl api-resources | grep -E 'gatewayclasses|gateways|httproutes'
   kubectl get gatewayclass
   kubectl get pods -A | grep -Ei 'envoy|tailscale'
   ```

   If these are missing on a local kind cluster, rerun
   `./scripts/bootstrap-kind.sh`. On remote clusters, install Kubernetes Gateway
   API, Envoy Gateway, and the Tailscale Kubernetes Operator using their
   official install docs, then rerun the checks. Ship's installer applies Ship's
   Gateway and dashboard resources; `scripts/bootstrap-kind.sh` handles the
   local kind add-ons.
7. Run the installer:

   ```sh
   curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | SHIP_DOMAIN=<my-domain> SHIP_ONBOARD=1 sh
   export PATH="$HOME/.local/bin:$PATH"
   ```

8. Capture the DNS line. If the installer prints:

   ```text
   manual dns: create *.<my-domain> as DNS-only CNAME/A record to <gateway-address>
   ```

   give the user that exact record and pause until DNS exists. Do not invent or
   create DNS records unless the user has explicitly provided DNS-provider
   credentials and asked you to manage DNS.
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
- cert-manager and a wildcard certificate issuer for custom-domain Gateway TLS

The installer expects the current `kubectl` context to be the cluster you want
Ship to use. For local kind, use a cluster named `ship`, which creates the
context `kind-ship`. Deployed service names are separate from the cluster name:
`k8s`, `e2e`, and `nitro-app` are services inside the cluster, not cluster
names.

## Bootstrap A Blank Local Cluster

Use the repository checkout when starting from no Kubernetes cluster:

```sh
TS_OAUTH_CLIENT_ID=<client-id> TS_OAUTH_CLIENT_SECRET=<client-secret> ./scripts/bootstrap-kind.sh
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

`install.sh` downloads the repository archive, builds `cmd/ship`, writes
`~/.config/ship/config.env`, and installs the binary at `~/.local/bin/ship`.

With `SHIP_ONBOARD=1`, it also runs:

```sh
scripts/bootstrap-kind.sh # when Tailscale OAuth env vars are present
deploy-system/deploy-domain.sh
deploy-system/deploy-dashboard.sh
```

That applies the default Tailscale Gateway, creates Cloudflare DNS when
`CLOUDFLARE_API_TOKEN` is set, and deploys the dashboard as the normal Ship
service named `k8s`.

The values you normally fill are:

```sh
SHIP_DOMAIN=mydomain.com
CLOUDFLARE_API_TOKEN=
TS_OAUTH_CLIENT_ID=
TS_OAUTH_CLIENT_SECRET=
# Optional; defaults to k8s.
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

For a dry-run deployment from any Dockerfile project:

```sh
ship --service demo --dry-run --json
```

## Public Internet Exposure

Ship defaults to Tailscale-only routing. To prepare Tailscale Funnel exposure:

```sh
cd deploy-system
./deploy-internet-gateway.sh
```

The script verifies the Tailscale IngressClass and prints the remaining manual
tailnet policy action for Funnel. Then deploy a service with a public Funnel
Ingress:

```sh
ship --service demo --exposure internet
```

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `ship: command not found` | Run `export PATH="$HOME/.local/bin:$PATH"` or add that line to your shell profile. |
| Installer prints `usage:` | Set `SHIP_DOMAIN=mydomain.com` or pass the domain with `sh -s -- mydomain.com`. |
| Dashboard does not load | Confirm the printed wildcard DNS record exists and points to the Gateway address. |
| Gateway has no address | Check the Tailscale Kubernetes Operator and Envoy Gateway installation for private routing; public exposure uses Tailscale Funnel Ingress. |
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

Remove the local CLI and config:

```sh
rm -f ~/.local/bin/ship
rm -rf ~/.config/ship
```

Remove the default cluster resources if this cluster is dedicated to Ship:

```sh
kubectl delete namespace ship-services
kubectl delete namespace ship-system
```

Remove the local kind cluster when it is dedicated to Ship:

```sh
kind delete cluster --name ship
```

If the namespaces contain resources you created outside Ship, delete only the
Ship-managed `Deployment`, `Service`, and `HTTPRoute` objects instead.
