#!/bin/sh
set -eu

repo="${SHIP_REPO:-gronxb/ship}"
ref="${SHIP_REF:-main}"
domain="${SHIP_DOMAIN:-${1:-}}"
prefix="${SHIP_IMAGE_PREFIX:-ship}"
onboard="${SHIP_ONBOARD:-0}"

if [ -z "$domain" ]; then
  printf 'usage: curl -fsSL https://raw.githubusercontent.com/%s/main/install.sh | SHIP_DOMAIN=mydomain.com SHIP_ONBOARD=1 sh\n' "$repo" >&2
  printf '   or: curl -fsSL https://raw.githubusercontent.com/%s/main/install.sh | sh -s -- mydomain.com\n' "$repo" >&2
  exit 2
fi

tmp="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT

bin_dir="${HOME}/.local/bin"
config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/ship"
mkdir -p "$bin_dir" "$config_dir"

curl -fsSL "https://github.com/${repo}/archive/refs/heads/${ref}.tar.gz" |
  tar -xz -C "$tmp" --strip-components=1

(cd "$tmp" && go build -o "$bin_dir/ship" ./cmd/ship)

cat > "$config_dir/config.env" <<EOF
SHIP_DOMAIN=$domain
SHIP_NAMESPACE=ship-services
SHIP_GATEWAY_NAMESPACE=ship-system
SHIP_GATEWAY_NAME=ship-tailscale
SHIP_INTERNET_GATEWAY_NAME=ship-internet
SHIP_DNS=manual
SHIP_IMAGE_PREFIX=$prefix
KIND_CLUSTER=ship
EOF

printf 'installed: %s/ship\n' "$bin_dir"
printf 'config: %s/config.env\n' "$config_dir"
printf 'next: export PATH="$HOME/.local/bin:$PATH"\n'

case "$onboard" in
  1 | true | yes)
    PATH="$bin_dir:$PATH"
    export PATH
    (cd "$tmp/deploy-system" && ./deploy-domain.sh && ./deploy-dashboard.sh)
    printf 'ready: ship CLI and https://k8s.%s\n' "$domain"
    ;;
  0 | false | no)
    ;;
  *)
    printf 'SHIP_ONBOARD must be 1 or 0\n' >&2
    exit 2
    ;;
esac
