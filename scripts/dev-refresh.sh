#!/bin/sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
ship_bin="${SHIP_BIN:-$HOME/.local/bin/ship}"

mkdir -p "$(dirname "$ship_bin")"
go build -o "$ship_bin" "$repo_root/cmd/ship"
printf 'ok: refreshed ship cli at %s\n' "$ship_bin"

cd "$repo_root/deploy-system"
SHIP_BIN="$ship_bin" SHIP_SOURCE_DIR="$repo_root" ./deploy-dashboard.sh
