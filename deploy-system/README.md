# Ship Deploy System

This directory creates the Gateway for Ship and can deploy the dashboard as the
normal `k8s` Ship service.

```sh
cp ../.env.example ../.env
$EDITOR ../.env
./deploy-domain.sh
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

Run one command before using the dashboard's "Expose to internet" action or
`ship --exposure internet`:

```sh
./deploy-internet-gateway.sh
```

The script verifies Tailscale Ingress support and prints the remaining manual
tailnet policy action. Public exposure is created as a Tailscale Funnel Ingress,
so local kind clusters do not need a public LoadBalancer.
