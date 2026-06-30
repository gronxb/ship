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
then tries Cloudflare DNS in `SHIP_DNS=auto`. If Cloudflare is not configured or
no DNS token is present, it prints the manual wildcard DNS record and exits
successfully.

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

Manual DNS mode:

```sh
SHIP_DNS=manual ./deploy-domain.sh
```

Strict Cloudflare mode:

```sh
export CLOUDFLARE_API_TOKEN=<token-with-zone-dns-edit>
export CLOUDFLARE_ZONE_ID=<optional-zone-id>
SHIP_DNS=cloudflare ./deploy-domain.sh
```

When `CLOUDFLARE_ZONE_ID` is not set, the token also needs Zone Read so Ship can
look up the zone id from `SHIP_DOMAIN`.

## Internet Gateway

`internet-gateway.yaml` is optional because local kind clusters usually cannot
assign a public LoadBalancer address.

```sh
./render-internet-gateway.sh | kubectl apply -f -
kubectl get gateway -n "$SHIP_GATEWAY_NAMESPACE" "$SHIP_INTERNET_GATEWAY_NAME"
```

After the Gateway has an address, create host-specific DNS records for services
you expose publicly.
