# Ship Deploy System

This directory creates the Gateway and dashboard route for Ship.

```sh
cp ../.env.example ../.env
$EDITOR ../.env
./deploy-domain.sh
```

`./deploy-domain.sh` reads `../.env` and `~/.config/ship/config.env`, renders the
`example.com` templates with your domain, applies the Tailscale Gateway, then
publishes `*.SHIP_DOMAIN` as a DNS-only Cloudflare record.

## Tailscale

The default Gateway uses:

- namespace: `SHIP_GATEWAY_NAMESPACE`
- Gateway: `SHIP_GATEWAY_NAME`
- dashboard host: `SHIP_DASHBOARD_HOST`
- Tailscale hostname annotation: `SHIP_GATEWAY_NAME`

Validate without applying:

```sh
./validate.sh
```

Render the final YAML:

```sh
./render.sh
```

## Internet Gateway

`internet-gateway.yaml` is optional because local kind clusters usually cannot
assign a public LoadBalancer address.

```sh
./render-internet-gateway.sh | kubectl apply -f -
kubectl get gateway -n "$SHIP_GATEWAY_NAMESPACE" "$SHIP_INTERNET_GATEWAY_NAME"
```

After the Gateway has an address, create host-specific DNS records for services
you expose publicly.
