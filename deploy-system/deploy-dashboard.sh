#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

repo_root="$(cd .. && pwd)"
ship_bin="${SHIP_BIN:-}"
if [ -z "$ship_bin" ]; then
  if command -v ship >/dev/null 2>&1; then
    ship_bin="$(command -v ship)"
  else
    ship_bin="$HOME/.local/bin/ship"
  fi
fi

service="${SHIP_DASHBOARD_SERVICE:-k8s}"
account="${SHIP_DASHBOARD_SERVICE_ACCOUNT:-k8s-dashboard}"
dashboard_host="$service.$SHIP_DOMAIN"
env_file="$(mktemp)"
cleanup() {
  rm -f "$env_file"
}
trap cleanup EXIT

kubectl create namespace "$SHIP_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: $account
  namespace: $SHIP_NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: $account
  namespace: $SHIP_NAMESPACE
rules:
  - apiGroups: ["gateway.networking.k8s.io"]
    resources: ["httproutes"]
    verbs: ["get", "list", "patch"]
  - apiGroups: ["networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["get", "list", "create", "patch", "update"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["pods", "pods/log"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: $account
  namespace: $SHIP_NAMESPACE
subjects:
  - kind: ServiceAccount
    name: $account
    namespace: $SHIP_NAMESPACE
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: $account
EOF

cat > "$env_file" <<EOF
SHIP_NAMESPACE=$SHIP_NAMESPACE
SHIP_DOMAIN=$SHIP_DOMAIN
SHIP_GATEWAY_NAMESPACE=$SHIP_GATEWAY_NAMESPACE
SHIP_GATEWAY_NAME=$SHIP_GATEWAY_NAME
SHIP_INTERNET_GATEWAY_NAME=$SHIP_INTERNET_GATEWAY_NAME
SHIP_DASHBOARD_HOST=$dashboard_host
EOF

"$ship_bin" \
  --service "$service" \
  --cwd "$repo_root/start-app" \
  --domain "$SHIP_DOMAIN" \
  --namespace "$SHIP_NAMESPACE" \
  --gateway-namespace "$SHIP_GATEWAY_NAMESPACE" \
  --gateway-name "$SHIP_GATEWAY_NAME" \
  --internet-gateway-name "$SHIP_INTERNET_GATEWAY_NAME" \
  --kind-cluster "${KIND_CLUSTER:-ship}" \
  --env-file "$env_file" \
  --service-account "$account"

printf 'ok: https://%s.%s is the Ship dashboard\n' "$service" "$SHIP_DOMAIN"
