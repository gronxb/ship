#!/bin/sh
set -eu

./validate.sh
. ./ship-env.sh

context="$(kubectl config current-context 2>/dev/null || true)"
if [ -z "$context" ]; then
  cat >&2 <<'EOF'
kubectl has no current context.
Configure kube access first, then rerun:
  export KUBECONFIG=/path/to/kubeconfig
  kubectl config use-context <context>
EOF
  exit 1
fi

./render.sh | kubectl apply -f -
kubectl rollout status deployment/ship-dashboard-proxy -n "$SHIP_GATEWAY_NAMESPACE" --timeout=180s

address=""
for _ in $(seq 1 60); do
  address="$(./gateway-address.sh "$SHIP_GATEWAY_NAMESPACE" "$SHIP_GATEWAY_NAME" 2>/dev/null || true)"
  [ -n "$address" ] && break
  sleep 5
done

if [ -z "$address" ]; then
  printf 'gateway did not publish an address within timeout\n' >&2
  exit 1
fi

./publish-cloudflare-dns.sh "$address"

printf 'ok: *.%s points to %s via Cloudflare DNS-only record\n' "$SHIP_DOMAIN" "$address"
