#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

secret="wildcard-$(printf '%s' "$SHIP_DOMAIN" | tr . -)-tls"

./render-internet-gateway.sh | kubectl apply -f -

printf '\n'
kubectl get gatewayclass internet
kubectl get gateway -n "$SHIP_GATEWAY_NAMESPACE" "$SHIP_INTERNET_GATEWAY_NAME" -o wide

address="$(./gateway-address.sh "$SHIP_GATEWAY_NAMESPACE" "$SHIP_INTERNET_GATEWAY_NAME" 2>/dev/null || true)"
if [ -z "$address" ]; then
  cat <<EOF

manual action: provide a public LoadBalancer address for Gateway $SHIP_GATEWAY_NAMESPACE/$SHIP_INTERNET_GATEWAY_NAME.
local kind clusters usually cannot do this; use Tailscale mode there or run this on a cloud cluster.
EOF
else
  cat <<EOF

manual action: create host-specific DNS records for public services under *.$SHIP_DOMAIN pointing to $address.
EOF
fi

if ! kubectl get secret "$secret" -n "$SHIP_GATEWAY_NAMESPACE" >/dev/null 2>&1; then
  cat <<EOF
manual action: create TLS Secret $SHIP_GATEWAY_NAMESPACE/$secret or configure cert-manager ClusterIssuer letsencrypt-dns01 to create it.
EOF
fi
