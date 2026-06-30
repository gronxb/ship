#!/bin/sh
set -eu

cd "$(dirname "$0")"
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

dns_error="$(mktemp)"
cleanup_dns_error() {
  rm -f "$dns_error"
}
trap cleanup_dns_error EXIT

case "${SHIP_DNS:-manual}" in
  manual)
    printf 'manual dns: create *.%s as DNS-only CNAME/A record to %s\n' "$SHIP_DOMAIN" "$address"
    ;;
  auto | cloudflare)
    if ./publish-cloudflare-dns.sh "$address" 2>"$dns_error"; then
      printf 'ok: *.%s points to %s via Cloudflare DNS-only record\n' "$SHIP_DOMAIN" "$address"
    elif [ "${SHIP_DNS:-manual}" = "cloudflare" ]; then
      cat "$dns_error" >&2
      exit 1
    else
      printf 'automatic dns skipped: %s\n' "$(sed -n '1p' "$dns_error")"
      printf 'manual dns: create *.%s as DNS-only CNAME/A record to %s\n' "$SHIP_DOMAIN" "$address"
    fi
    ;;
  *)
    printf 'SHIP_DNS must be auto, manual, or cloudflare\n' >&2
    exit 2
    ;;
esac
