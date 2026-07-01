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
real_go="$(command -v go)"
mkdir -p "$fakebin" "$home"

cat > "$fakebin/curl" <<'SH'
#!/bin/sh
printf 'curl %s\n' "$*" >> "$SHIP_TEST_LOG"
last=""
for arg in "$@"; do
  last="$arg"
done
case "$last" in
  *api.github.com*/releases/latest) printf '{"tag_name":"v2.0.0"}\n' ;;
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
    printf 'create secret %s %s\n' "$3" "$4" >> "$SHIP_TEST_LOG"
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
  'get secret')
    exit 1
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

cat > "$fakebin/helm" <<'SH'
#!/bin/sh
printf 'helm %s\n' "$*" >> "$SHIP_TEST_LOG"
case "$1" in
  status)
    exit 1
    ;;
esac
SH

cat > "$fakebin/go" <<'SH'
#!/bin/sh
printf 'go %s\n' "$*" >> "$SHIP_TEST_LOG"
exec "$SHIP_TEST_REAL_GO" "$@"
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
SHIP_TEST_REAL_GO="$real_go" \
sh -c 'curl -fsSL https://raw.githubusercontent.com/gronxb/ship/main/install.sh | sh' \
  > "$work/stdout" \
  2> "$work/stderr"

PATH="$home/.local/bin:$PATH" HOME="$home" ship --help > "$work/help"

grep -Fq 'installed:' "$work/stdout"
grep -Fq 'next: fill .env, then run: ship install' "$work/stdout"
if [ -e "$home/.config/ship/config.env" ]; then
  printf 'installer should not create config before ship install\n' >&2
  exit 1
fi
if grep -iq 'cloudflare' "$work/stdout" "$work/stderr"; then
  cat "$work/stdout"
  cat "$work/stderr" >&2
  printf 'default onboarding must not mention Cloudflare\n' >&2
  exit 1
fi
grep -Fq 'Usage:' "$work/help"

cloudflare_home="$work/cloudflare-home"
mkdir -p "$cloudflare_home"
cloudflare_log="$work/cloudflare-commands.log"
cloudflare_env="$work/cloudflare.env"
cat > "$cloudflare_env" <<'EOF'
SHIP_DOMAIN=example.com
SHIP_DRY_RUN=1
SHIP_DASHBOARD_SERVICE=ops
CLOUDFLARE_API_TOKEN=test-token
TAILSCALE_CLIENT_ID=test-client
TAILSCALE_CLIENT_SECRET=test-secret
SHIP_BIN=__SHIP_BIN__
EOF
sed "s#__SHIP_BIN__#$home/.local/bin/ship#" "$cloudflare_env" > "$cloudflare_env.tmp"
mv "$cloudflare_env.tmp" "$cloudflare_env"

PATH="$home/.local/bin:$fakebin:$PATH" \
HOME="$cloudflare_home" \
SHIP_TEST_LOG="$cloudflare_log" \
SHIP_SOURCE_DIR="$root" \
ship install --env-file "$cloudflare_env" \
  > "$work/cloudflare-stdout" \
  2> "$work/cloudflare-stderr"

grep -Fq 'ready: ship install complete at https://ops.example.com' "$work/cloudflare-stdout"
grep -Fq 'SHIP_DNS=cloudflare' "$cloudflare_home/.config/ship/config.env"
grep -Fq 'SHIP_DASHBOARD_SERVICE=ops' "$cloudflare_home/.config/ship/config.env"
grep -Fq 'kind create cluster --name ship' "$cloudflare_log"
grep -Fq 'helm upgrade --install tailscale-operator tailscale/tailscale-operator' "$cloudflare_log"
grep -Fq 'helm upgrade --install cert-manager oci://quay.io/jetstack/charts/cert-manager' "$cloudflare_log"
grep -Fq -- '--set config.enableGatewayAPI=true' "$cloudflare_log"
grep -Fq -- '--dns01-recursive-nameservers=1.1.1.1:53\,8.8.8.8:53' "$cloudflare_log"
grep -Fq 'create secret generic cloudflare-api-token-secret' "$cloudflare_log"
grep -Fq 'ClusterIssuer' "$cloudflare_log"
grep -Fq 'wait --for=condition=Ready certificate/wildcard-example-com-tls -n ship-system --timeout=10m' "$cloudflare_log"
if grep -Fq 'create secret tls wildcard-example-com-tls' "$cloudflare_log"; then
  printf 'ship install must not create a self-signed wildcard TLS secret by default\n' >&2
  exit 1
fi
grep -Fq 'dry-run: *.example.com CNAME ship-tailscale.tailnet.test proxied=false' "$work/cloudflare-stdout"
grep -Fq 'docker build -f' "$cloudflare_log"
grep -Fq -- '-t ship/ops:' "$cloudflare_log"
grep -Fq 'kind load docker-image --name ship ship/ops:' "$cloudflare_log"
grep -Fq 'kubectl rollout status deployment/ops -n ship-services --timeout=180s' "$cloudflare_log"

PATH="$home/.local/bin:$fakebin:$PATH" \
HOME="$cloudflare_home" \
SHIP_TEST_LOG="$cloudflare_log" \
SHIP_SOURCE_DIR="$root" \
ship uninstall --env-file "$cloudflare_env" --dry-run \
  > "$work/uninstall-stdout" \
  2> "$work/uninstall-stderr"
grep -Fq 'delete Cloudflare wildcard DNS for *.example.com' "$work/uninstall-stdout"
grep -Fq 'delete Tailscale devices named ship-tailscale* and tailscale-operator*' "$work/uninstall-stdout"
grep -Fq 'kind delete cluster --name ship' "$work/uninstall-stdout"

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
strict_root="$work/strict-root"
mkdir -p "$strict_home/.config/ship"
mkdir -p "$strict_root"
cp -R "$root/deploy-system" "$strict_root/deploy-system"
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
"$strict_root/deploy-system/deploy-domain.sh" \
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
if ! grep -Fq "missing CLOUDFLARE_API_TOKEN for Let's Encrypt wildcard TLS" "$work/strict-stderr"; then
  cat "$work/strict-stdout"
  cat "$work/strict-stderr" >&2
  exit 1
fi

printf 'onboarding-smoke: ok\n'
