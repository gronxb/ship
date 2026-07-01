#!/bin/sh
set -eu

cluster="${KIND_CLUSTER:-ship}"
context="${KIND_CONTEXT:-kind-$cluster}"
envoy_version="${ENVOY_GATEWAY_VERSION:-v1.8.2}"
tailscale_client_id="${TS_OAUTH_CLIENT_ID:-${TAILSCALE_OAUTH_CLIENT_ID:-}}"
tailscale_client_secret="${TS_OAUTH_CLIENT_SECRET:-${TAILSCALE_OAUTH_CLIENT_SECRET:-}}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'missing required command: %s\n' "$1" >&2
    exit 127
  }
}

need kind
need kubectl
need helm

if kind get clusters | grep -Fxq "$cluster"; then
  printf 'ok: kind cluster %s already exists\n' "$cluster"
else
  kind create cluster --name "$cluster"
fi

kubectl config use-context "$context"
kubectl get namespace >/dev/null

helm upgrade --install eg oci://docker.io/envoyproxy/gateway-helm \
  --version "$envoy_version" \
  -n envoy-gateway-system \
  --create-namespace

kubectl wait --timeout=5m -n envoy-gateway-system \
  deployment/envoy-gateway \
  --for=condition=Available

if helm status tailscale-operator -n tailscale >/dev/null 2>&1; then
  printf 'ok: tailscale-operator already installed\n'
elif [ -n "$tailscale_client_id" ] && [ -n "$tailscale_client_secret" ]; then
  helm repo add tailscale https://pkgs.tailscale.com/helmcharts >/dev/null 2>&1 || true
  helm repo update tailscale
  helm upgrade --install tailscale-operator tailscale/tailscale-operator \
    --namespace=tailscale \
    --create-namespace \
    --set-string oauth.clientId="$tailscale_client_id" \
    --set-string oauth.clientSecret="$tailscale_client_secret" \
    --wait
else
  cat >&2 <<'EOF'
missing Tailscale OAuth credentials.

Create a Tailscale OAuth client for the Kubernetes Operator, then rerun with:
  TS_OAUTH_CLIENT_ID=<client-id> TS_OAUTH_CLIENT_SECRET=<client-secret> ./scripts/bootstrap-kind.sh

Aliases also accepted:
  TAILSCALE_OAUTH_CLIENT_ID
  TAILSCALE_OAUTH_CLIENT_SECRET
EOF
  exit 2
fi

kubectl get crd gateways.gateway.networking.k8s.io >/dev/null
kubectl get crd httproutes.gateway.networking.k8s.io >/dev/null
kubectl get crd envoyproxies.gateway.envoyproxy.io >/dev/null
kubectl get pods -n tailscale >/dev/null

printf 'ready: kind cluster %s with Envoy Gateway and Tailscale Operator\n' "$cluster"
