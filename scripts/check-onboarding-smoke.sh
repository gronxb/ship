#!/bin/sh
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
cleanup() {
  rm -rf "$work"
}
trap cleanup EXIT

repo_tar="$work/ship.tgz"
tar \
  --exclude .git \
  --exclude .omo \
  --exclude .omx \
  --exclude start-app/node_modules \
  --exclude start-app/dist \
  -czf "$repo_tar" \
  -C "$(dirname "$root")" \
  "$(basename "$root")"

fakebin="$work/bin"
home="$work/home"
log="$work/commands.log"
mkdir -p "$fakebin" "$home"

cat > "$fakebin/curl" <<'SH'
#!/bin/sh
printf 'curl %s\n' "$*" >> "$SHIP_TEST_LOG"
last=""
for arg in "$@"; do
  last="$arg"
done
case "$last" in
  *install.sh) cat "$SHIP_TEST_INSTALL" ;;
  *) cat "$SHIP_TEST_TARBALL" ;;
esac
SH

cat > "$fakebin/kubectl" <<'SH'
#!/bin/sh
printf 'kubectl %s\n' "$*" >> "$SHIP_TEST_LOG"
case "$1 $2" in
  'config current-context')
    printf 'kind-ship\n'
    ;;
  'kustomize .')
    cat <<'YAML'
apiVersion: v1
kind: Namespace
metadata:
  name: ship-system
---
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: EnvoyProxy
metadata:
  name: tailscale-proxy
  namespace: ship-system
spec:
  provider:
    type: Kubernetes
    kubernetes:
      envoyService:
        type: LoadBalancer
        loadBalancerClass: tailscale
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: ship-tailscale
  namespace: ship-system
spec:
  listeners:
    - hostname: "*.example.com"
YAML
    ;;
  'get gateway')
    printf '{"status":{"addresses":[{"value":"ship-tailscale.tailnet.test"}]}}\n'
    ;;
  'create namespace')
    printf 'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n' "$3"
    ;;
  'create secret')
    env_file=""
    for arg in "$@"; do
      case "$arg" in
        --from-env-file=*) env_file="${arg#--from-env-file=}" ;;
      esac
    done
    if [ -n "$env_file" ]; then
      printf 'secret env file %s\n' "$env_file" >> "$SHIP_TEST_LOG"
      sed 's/^/secret env /' "$env_file" >> "$SHIP_TEST_LOG"
    fi
    printf 'apiVersion: v1\nkind: Secret\nmetadata:\n  name: %s\n' "$4"
    ;;
  'rollout status')
    printf 'deployment %s rolled out\n' "$3"
    ;;
  'apply -f')
    cat >> "$SHIP_TEST_LOG"
    printf 'applied\n'
    ;;
  *)
    printf '{}\n'
    ;;
esac
SH

cat > "$fakebin/docker" <<'SH'
#!/bin/sh
printf 'docker %s\n' "$*" >> "$SHIP_TEST_LOG"
SH

cat > "$fakebin/kind" <<'SH'
#!/bin/sh
printf 'kind %s\n' "$*" >> "$SHIP_TEST_LOG"
SH

cat > "$fakebin/go" <<'SH'
#!/bin/sh
printf 'go %s\n' "$*" >> "$SHIP_TEST_LOG"
case "$1" in
  build)
    out=""
    while [ "$#" -gt 0 ]; do
      case "$1" in
        -o)
          out="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [ -z "$out" ]; then
      printf 'fake go build requires -o\n' >&2
      exit 2
    fi
    mkdir -p "$(dirname "$out")"
    cat > "$out" <<'BIN'
#!/bin/sh
printf 'usage: ship --service <name> [--cwd DIR] [--port PORT] [--dry-run] [--json]\n'
BIN
    chmod +x "$out"
    ;;
  *)
    printf 'fake go only supports build\n' >&2
    exit 2
    ;;
esac
SH

cat > "$fakebin/ship" <<'SH'
#!/bin/sh
printf 'ship %s\n' "$*" >> "$SHIP_TEST_LOG"
printf 'ship env SHIP_IMAGE_PREFIX=%s\n' "${SHIP_IMAGE_PREFIX:-}" >> "$SHIP_TEST_LOG"
printf 'ship env KIND_CLUSTER=%s\n' "${KIND_CLUSTER:-}" >> "$SHIP_TEST_LOG"
printf 'ship env REGISTRY=%s\n' "${REGISTRY:-}" >> "$SHIP_TEST_LOG"
SH

chmod +x "$fakebin"/*

PATH="$fakebin:$PATH" \
HOME="$home" \
SHIP_TEST_INSTALL="$root/install.sh" \
SHIP_TEST_TARBALL="$repo_tar" \
SHIP_TEST_LOG="$log" \
SHIP_DOMAIN=example.com \
SHIP_DASHBOARD_HOST=stale.example.net \
SHIP_ONBOARD=1 \
sh -c 'curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh' \
  > "$work/stdout" \
  2> "$work/stderr"

PATH="$home/.local/bin:$PATH" HOME="$home" ship --help > "$work/help"

grep -Fq 'ready: ship CLI and https://k8s.example.com' "$work/stdout"
grep -Fq 'manual dns: create *.example.com' "$work/stdout"
grep -Fq 'SHIP_DNS=manual' "$home/.config/ship/config.env"
if grep -iq 'cloudflare' "$work/stdout" "$work/stderr"; then
  cat "$work/stdout"
  cat "$work/stderr" >&2
  printf 'default onboarding must not mention Cloudflare\n' >&2
  exit 1
fi
grep -Fq 'docker build' "$log"
grep -Fq 'kind load docker-image --name ship ship/k8s:' "$log"
grep -Fq 'kubectl rollout status deployment/k8s -n ship-services --timeout=180s' "$log"
grep -Fq 'verbs: ["get", "list", "patch"]' "$log"
grep -Fq 'secret env SHIP_DASHBOARD_HOST=k8s.example.com' "$log"
grep -Fq 'usage: ship' "$work/help"

env_home="$work/env-home"
mkdir -p "$env_home/.config/ship"
cat > "$env_home/.config/ship/config.env" <<'EOF'
SHIP_DOMAIN=example.com
SHIP_IMAGE_PREFIX=bad-prefix
KIND_CLUSTER=bad-kind
REGISTRY=bad.registry.test
EOF

PATH="$fakebin:$PATH" \
HOME="$env_home" \
SHIP_TEST_LOG="$log" \
SHIP_IMAGE_PREFIX=good-prefix \
KIND_CLUSTER=good-kind \
REGISTRY=good.registry.test \
"$root/deploy-system/deploy-dashboard.sh" \
  > "$work/env-stdout" \
  2> "$work/env-stderr"

grep -Fq 'ship env SHIP_IMAGE_PREFIX=good-prefix' "$log"
grep -Fq 'ship env KIND_CLUSTER=good-kind' "$log"
grep -Fq 'ship env REGISTRY=good.registry.test' "$log"
grep -Fq -- '--kind-cluster good-kind' "$log"

strict_home="$work/strict-home"
mkdir -p "$strict_home/.config/ship"
cat > "$strict_home/.config/ship/config.env" <<'EOF'
SHIP_DOMAIN=example.com
SHIP_DNS=auto
SHIP_DASHBOARD_HOST=stale.example.net
EOF

set +e
PATH="$fakebin:$PATH" \
HOME="$strict_home" \
WRANGLER="$work/missing-wrangler" \
SHIP_TEST_LOG="$log" \
SHIP_DNS=cloudflare \
"$root/deploy-system/deploy-domain.sh" \
  > "$work/strict-stdout" \
  2> "$work/strict-stderr"
strict_status=$?
set -e

if [ "$strict_status" -eq 0 ]; then
  cat "$work/strict-stdout"
  cat "$work/strict-stderr" >&2
  printf 'expected SHIP_DNS=cloudflare to override config SHIP_DNS=auto\n' >&2
  exit 1
fi
grep -Fq 'missing CLOUDFLARE_API_TOKEN with Zone DNS Edit' "$work/strict-stderr"

printf 'onboarding-smoke: ok\n'
