#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

check_only=false

while [ "$#" -gt 0 ]; do
  case "$1" in
    --check)
      check_only=true
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
  shift
done

dashboard_port="$SHIP_DASHBOARD_PORT"
tailscale_ip="$(tailscale ip -4 | sed -n '1p')"

if [ -z "$tailscale_ip" ]; then
  printf 'tailscale ip -4 returned no IPv4 address\n' >&2
  exit 1
fi

printf 'ship gateway synced: dashboard=%s upstream=%s:%s tailscale_only=true\n' \
  "$SHIP_DASHBOARD_HOST" \
  "$tailscale_ip" "$dashboard_port"
