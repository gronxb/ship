#!/bin/sh
set -eu

repo="${SHIP_REPO:-gronxb/ship}"
ref="${SHIP_REF:-main}"
domain="${SHIP_DOMAIN:-${1:-}}"
prefix="${SHIP_IMAGE_PREFIX:-ship}"
onboard="${SHIP_ONBOARD:-0}"
dashboard_service="${SHIP_DASHBOARD_SERVICE:-k8s}"
dns="${SHIP_DNS:-}"
if [ -z "$dns" ]; then
  if [ -n "${CLOUDFLARE_API_TOKEN:-${CF_API_TOKEN:-}}" ]; then
    dns="cloudflare"
  else
    dns="manual"
  fi
fi

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

{
  printf 'SHIP_DOMAIN=%s\n' "$domain"
  if [ "$dns" != "manual" ]; then
    printf 'SHIP_DNS=%s\n' "$dns"
  fi
  if [ "$dashboard_service" != "k8s" ]; then
    printf 'SHIP_DASHBOARD_SERVICE=%s\n' "$dashboard_service"
  fi
  if [ "$prefix" != "ship" ]; then
    printf 'SHIP_IMAGE_PREFIX=%s\n' "$prefix"
  fi
} > "$config_dir/config.env"

printf 'installed: %s/ship\n' "$bin_dir"
printf 'config: %s/config.env\n' "$config_dir"
printf 'next: export PATH="$HOME/.local/bin:$PATH"\n'

case "$onboard" in
  1 | true | yes)
    PATH="$bin_dir:$PATH"
    export PATH
    if [ -n "${TS_OAUTH_CLIENT_ID:-${TAILSCALE_OAUTH_CLIENT_ID:-}}" ] &&
      [ -n "${TS_OAUTH_CLIENT_SECRET:-${TAILSCALE_OAUTH_CLIENT_SECRET:-}}" ]; then
      (cd "$tmp" && ./scripts/bootstrap-kind.sh)
    fi
    (cd "$tmp/deploy-system" && ./deploy-domain.sh && ./deploy-dashboard.sh)
    printf 'ready: ship CLI and https://%s.%s\n' "$dashboard_service" "$domain"
    ;;
  0 | false | no)
    ;;
  *)
    printf 'SHIP_ONBOARD must be 1 or 0\n' >&2
    exit 2
    ;;
esac
