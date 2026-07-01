#!/bin/sh
set -eu

repo="${SHIP_REPO:-gronxb/ship}"
ref="${SHIP_REF:-latest}"

latest_release_tag() {
  curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" |
    sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' |
    sed -n '1p'
}

archive_kind() {
  case "$1" in
    v*)
      printf 'tags'
      ;;
    *)
      printf 'heads'
      ;;
  esac
}

if [ "$ref" = "latest" ]; then
  ref="$(latest_release_tag)"
  if [ -z "$ref" ]; then
    printf 'error: no latest GitHub Release found for %s\n' "$repo" >&2
    printf 'hint: create a v* release tag, or set SHIP_REF=main to install unreleased code.\n' >&2
    exit 1
  fi
fi

tmp="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT

bin_dir="${HOME}/.local/bin"
mkdir -p "$bin_dir"

kind="$(archive_kind "$ref")"
version="$ref"

curl -fsSL "https://github.com/${repo}/archive/refs/${kind}/${ref}.tar.gz" |
  tar -xz -C "$tmp" --strip-components=1

(cd "$tmp" && go build -ldflags "-X main.version=$version -X main.sourceRepo=$repo -X main.sourceRef=$ref" -o "$bin_dir/ship" ./cmd/ship)

printf 'installed: %s/ship\n' "$bin_dir"
printf 'version: %s\n' "$version"
printf 'next: export PATH="$HOME/.local/bin:$PATH"\n'
printf 'next: fill .env, then run: ship install\n'
