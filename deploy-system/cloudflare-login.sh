#!/bin/sh
set -eu

wrangler="${WRANGLER:-}"
if [ -z "$wrangler" ] && command -v wrangler >/dev/null 2>&1; then
  wrangler="$(command -v wrangler)"
fi
if [ -z "$wrangler" ]; then
  printf 'wrangler not found. Set WRANGLER=/path/to/wrangler and retry.\n' >&2
  exit 1
fi

exec "$wrangler" login \
  --scopes "account:read user:read workers:write workers_scripts:write workers_routes:write zone:read"
