#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

if [ -n "${SHIP_TLS_CERT_FILE:-}" ] || [ -n "${SHIP_TLS_KEY_FILE:-}" ]; then
  printf 'ok: custom TLS files set; skipping cert-manager setup\n'
  exit 0
fi

if [ "${SHIP_TLS_ALLOW_SELF_SIGNED:-}" = "1" ]; then
  printf 'ok: self-signed TLS explicitly enabled; skipping cert-manager setup\n'
  exit 0
fi

token="${CLOUDFLARE_API_TOKEN:-${CF_API_TOKEN:-}}"
if [ -z "$token" ]; then
  cat >&2 <<'EOF'
missing CLOUDFLARE_API_TOKEN for Let's Encrypt wildcard TLS.
Set CLOUDFLARE_API_TOKEN, or set SHIP_TLS_CERT_FILE and SHIP_TLS_KEY_FILE.
Self-signed certificates are disabled by default; set SHIP_TLS_ALLOW_SELF_SIGNED=1 only for local experiments.
EOF
  exit 2
fi

command -v helm >/dev/null 2>&1 || {
  printf 'missing required command: helm\n' >&2
  exit 127
}

version="${CERT_MANAGER_VERSION:-v1.20.3}"
email="${SHIP_ACME_EMAIL:-admin@$SHIP_DOMAIN}"

helm upgrade --install cert-manager oci://quay.io/jetstack/charts/cert-manager \
  --version "$version" \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true \
  --set config.enableGatewayAPI=true \
  --set 'extraArgs={--dns01-recursive-nameservers-only,--dns01-recursive-nameservers=1.1.1.1:53\,8.8.8.8:53}' \
  --wait

kubectl create secret generic cloudflare-api-token-secret \
  -n cert-manager \
  --from-literal=api-token="$token" \
  --dry-run=client \
  -o yaml | kubectl apply -f -

cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-dns01
spec:
  acme:
    email: $email
    server: https://acme-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: letsencrypt-dns01-account-key
    solvers:
      - dns01:
          cloudflare:
            apiTokenSecretRef:
              name: cloudflare-api-token-secret
              key: api-token
EOF

kubectl wait --for=condition=Ready clusterissuer/letsencrypt-dns01 --timeout=2m
printf 'ok: cert-manager wildcard issuer letsencrypt-dns01 is ready\n'
