#!/bin/sh
set -eu

repo="${SHIP_REPO:-gronxb/ship}"
ref="${SHIP_REF:-main}"

tmp="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT

bin_dir="${HOME}/.local/bin"
mkdir -p "$bin_dir"

curl -fsSL "https://github.com/${repo}/archive/refs/heads/${ref}.tar.gz" |
  tar -xz -C "$tmp" --strip-components=1

(cd "$tmp" && go build -ldflags "-X main.sourceRepo=$repo -X main.sourceRef=$ref" -o "$bin_dir/ship" ./cmd/ship)

printf 'installed: %s/ship\n' "$bin_dir"
printf 'next: export PATH="$HOME/.local/bin:$PATH"\n'
printf 'next: fill .env, then run: ship install\n'
