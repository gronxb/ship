#!/bin/sh
set -eu

if [ -n "${CLOUDFLARE_API_TOKEN:-${CF_API_TOKEN:-}}" ]; then
  printf 'ok: Cloudflare API token is set\n'
  exit 0
fi

cat >&2 <<'EOF'
Cloudflare automatic DNS and internet exposure need an API token with:
  - Zone DNS Edit
  - Zone Read, unless CLOUDFLARE_ZONE_ID is set
  - Account Cloudflare Tunnel Edit, so ship install can create the tunnel

Set it before strict Cloudflare mode:

  export CLOUDFLARE_API_TOKEN=<token-with-zone-dns-and-tunnel-edit>
  export CLOUDFLARE_ZONE_ID=<optional-zone-id>
  SHIP_DNS=cloudflare ./deploy-domain.sh

Without a token, keep SHIP_DNS=auto or SHIP_DNS=manual and create the printed wildcard DNS record at your DNS provider.
EOF
exit 1
