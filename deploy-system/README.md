# Ship Deploy System

This directory creates the Gateway for Ship and can deploy the dashboard as the
normal `k8s` Ship service.

```sh
cp ../.env.example ../.env
$EDITOR ../.env
./deploy-domain.sh
./deploy-cloudflare-tunnel.sh
./deploy-dashboard.sh
```

`./deploy-domain.sh` reads `../.env` and `~/.config/ship/config.env`, renders the
`example.com` Gateway template with your domain, applies the Tailscale Gateway,
installs cert-manager with Gateway API support, creates a Let's Encrypt
Cloudflare DNS-01 ClusterIssuer, configures public recursive resolvers for the
DNS-01 self-check, waits for the wildcard TLS certificate, then creates the
Cloudflare wildcard DNS record when `SHIP_DNS=cloudflare` and
`CLOUDFLARE_API_TOKEN` are set. In manual mode it prints the record to create.
Self-signed certificates are disabled by default; use `SHIP_TLS_CERT_FILE` and
`SHIP_TLS_KEY_FILE` for a custom wildcard certificate.

`./deploy-dashboard.sh` applies the minimal dashboard RBAC and runs `ship` against
`../start-app`, so `https://k8s.$SHIP_DOMAIN` is backed by an in-cluster
Deployment/Service/HTTPRoute instead of a local dev proxy.

`./deploy-cloudflare-tunnel.sh` creates or reuses a Cloudflare Tunnel, stores the
account and tunnel IDs in the Ship config, and runs an in-cluster `cloudflared`
connector. It does not make services public by itself; dashboard and CLI
exposure actions add and remove specific hostnames later.

## Tailscale

The default Gateway uses:

- namespace: `SHIP_GATEWAY_NAMESPACE`
- Gateway: `SHIP_GATEWAY_NAME`
- Tailscale hostname annotation: `SHIP_GATEWAY_NAME`

Validate without applying:

```sh
./validate.sh
```

Render the final YAML:

```sh
./render.sh
```

Default DNS mode:

```sh
./deploy-domain.sh
```

The command prints:

```text
manual dns: create *.$SHIP_DOMAIN as DNS-only CNAME/A record to <gateway-address>
```

Cloudflare DNS mode:

```sh
SHIP_DNS=cloudflare CLOUDFLARE_API_TOKEN=<token> ./deploy-domain.sh
```

The same wildcard certificate is used by the default dashboard service
(`k8s.$SHIP_DOMAIN`) and later `ship --service ...` routes under
`*.$SHIP_DOMAIN`.

## Internet Exposure

`ship install` runs `./deploy-cloudflare-tunnel.sh` automatically after the
Tailscale Gateway and wildcard DNS are ready. The default wildcard DNS record
stays DNS-only and points at the Tailscale Gateway. Public exposure creates a
specific proxied Cloudflare CNAME for the service hostname and routes it through
the tunnel to the in-cluster service.

```sh
ship --service demo --exposure internet
```

Switching the same service back to Tailscale removes the tunnel hostname and
specific CNAME, so the wildcard Tailscale route becomes active again.
