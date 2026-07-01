#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

if [ -n "${SHIP_TLS_CERT_FILE:-}" ] || [ "${SHIP_TLS_ALLOW_SELF_SIGNED:-}" = "1" ]; then
  printf 'ok: skipping cert-manager certificate wait for explicit TLS mode\n'
  exit 0
fi

secret="${SHIP_TLS_SECRET:-wildcard-$(printf '%s' "$SHIP_DOMAIN" | tr . -)-tls}"
timeout="${SHIP_TLS_CERT_TIMEOUT:-10m}"

if ! kubectl get crd certificates.cert-manager.io >/dev/null 2>&1; then
  printf 'cert-manager Certificate CRD is missing\n' >&2
  exit 2
fi

found=""
for _ in $(seq 1 30); do
  if kubectl get certificate "$secret" -n "$SHIP_GATEWAY_NAMESPACE" >/dev/null 2>&1; then
    found=1
    break
  fi
  sleep 2
done

if [ -z "$found" ]; then
  printf 'cert-manager did not create Certificate %s/%s\n' "$SHIP_GATEWAY_NAMESPACE" "$secret" >&2
  kubectl get gateway "$SHIP_GATEWAY_NAME" -n "$SHIP_GATEWAY_NAMESPACE" -o yaml >&2 || true
  exit 1
fi

if ! kubectl wait --for=condition=Ready "certificate/$secret" -n "$SHIP_GATEWAY_NAMESPACE" --timeout="$timeout"; then
  kubectl describe certificate "$secret" -n "$SHIP_GATEWAY_NAMESPACE" >&2 || true
  exit 1
fi

printf 'ok: trusted TLS certificate %s/%s is ready\n' "$SHIP_GATEWAY_NAMESPACE" "$secret"
