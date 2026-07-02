#!/bin/sh
set -eu

repo="${SHIP_REPO:-gronxb/ship}"
ref="${SHIP_REF:-latest}"

latest_release_tag() {
  curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" |
    sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' |
    sed -n '1p'
}

release_os() {
  case "$(uname -s)" in
    Darwin)
      printf 'darwin'
      ;;
    Linux)
      printf 'linux'
      ;;
    *)
      printf 'error: unsupported OS for release install: %s\n' "$(uname -s)" >&2
      exit 1
      ;;
  esac
}

release_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      printf 'amd64'
      ;;
    arm64|aarch64)
      printf 'arm64'
      ;;
    *)
      printf 'error: unsupported architecture for release install: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac
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

version="$ref"

case "$ref" in
  v*)
    os="$(release_os)"
    arch="$(release_arch)"
    asset="ship_${ref}_${os}_${arch}.tar.gz"
    curl -fsSL "https://github.com/${repo}/releases/download/${ref}/${asset}" |
      tar -xz -C "$bin_dir" ship
    chmod +x "$bin_dir/ship"
    ;;
  *)
    kind="$(archive_kind "$ref")"
    curl -fsSL "https://github.com/${repo}/archive/refs/${kind}/${ref}.tar.gz" |
      tar -xz -C "$tmp" --strip-components=1
    (cd "$tmp" && go build -ldflags "-X main.version=$version -X main.sourceRepo=$repo -X main.sourceRef=$ref" -o "$bin_dir/ship" ./cmd/ship)
    ;;
esac

printf 'installed: %s/ship\n' "$bin_dir"
printf 'version: %s\n' "$version"
printf 'next: export PATH="$HOME/.local/bin:$PATH"\n'
printf 'next: fill .env, then run: ship install\n'
