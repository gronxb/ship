#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

secret="${SHIP_TLS_SECRET:-wildcard-$(printf '%s' "$SHIP_DOMAIN" | tr . -)-tls}"

if kubectl get secret "$secret" -n "$SHIP_GATEWAY_NAMESPACE" >/dev/null 2>&1; then
  printf 'ok: TLS secret %s/%s exists\n' "$SHIP_GATEWAY_NAMESPACE" "$secret"
  exit 0
fi

apply_secret() {
  cert_file=$1
  key_file=$2

  kubectl create secret tls "$secret" \
    -n "$SHIP_GATEWAY_NAMESPACE" \
    --cert="$cert_file" \
    --key="$key_file" \
    --dry-run=client \
    -o yaml | kubectl apply -f -
}

if [ -n "${SHIP_TLS_CERT_FILE:-}" ] || [ -n "${SHIP_TLS_KEY_FILE:-}" ]; then
  if [ -z "${SHIP_TLS_CERT_FILE:-}" ] || [ -z "${SHIP_TLS_KEY_FILE:-}" ]; then
    printf 'SHIP_TLS_CERT_FILE and SHIP_TLS_KEY_FILE must be set together\n' >&2
    exit 2
  fi
  apply_secret "$SHIP_TLS_CERT_FILE" "$SHIP_TLS_KEY_FILE"
  printf 'ok: TLS secret %s/%s applied from cert files\n' "$SHIP_GATEWAY_NAMESPACE" "$secret"
  exit 0
fi

if ! command -v openssl >/dev/null 2>&1; then
  printf 'missing openssl; set SHIP_TLS_CERT_FILE and SHIP_TLS_KEY_FILE instead\n' >&2
  exit 1
fi

tmp="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT

cat > "$tmp/openssl.cnf" <<EOF
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = *.$SHIP_DOMAIN

[v3_req]
subjectAltName = DNS:*.$SHIP_DOMAIN,DNS:$SHIP_DOMAIN
EOF

openssl req \
  -x509 \
  -nodes \
  -newkey rsa:2048 \
  -days "${SHIP_TLS_SELF_SIGNED_DAYS:-30}" \
  -keyout "$tmp/tls.key" \
  -out "$tmp/tls.crt" \
  -config "$tmp/openssl.cnf" \
  >/dev/null 2>&1

apply_secret "$tmp/tls.crt" "$tmp/tls.key"
printf 'ok: TLS secret %s/%s created with self-signed wildcard cert\n' "$SHIP_GATEWAY_NAMESPACE" "$secret"
