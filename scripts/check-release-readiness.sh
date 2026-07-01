#!/bin/sh
set -eu

fail() {
  printf 'release-readiness: %s\n' "$1" >&2
  exit 1
}

require_file() {
  test -f "$1" || fail "missing $1"
}

require_dir() {
  test -d "$1" || fail "missing $1"
}

require_text() {
  file="$1"
  text="$2"
  grep -Fq "$text" "$file" || fail "missing '$text' in $file"
}

require_file LICENSE
require_file README.md
require_file CONTRIBUTING.md
require_file SECURITY.md
require_file CODE_OF_CONDUCT.md
require_file CHANGELOG.md
require_file Makefile
require_file .github/workflows/ci.yml
require_file .github/dependabot.yml
require_file .github/PULL_REQUEST_TEMPLATE.md
require_file .github/ISSUE_TEMPLATE/bug_report.yml
require_file .github/ISSUE_TEMPLATE/feature_request.yml
require_file .env.example
require_file install.sh
require_file scripts/bootstrap-kind.sh
require_file deploy-system/validate.sh
require_file deploy-system/deploy-dashboard.sh
require_file start-app/Dockerfile
require_file start-app/package.json
require_file start-app/pnpm-lock.yaml
require_file go.mod
require_dir cmd/ship
require_dir internal/deploy
require_dir start-app/src

require_text README.md "Quick Start"
require_text README.md "For humans"
require_text README.md "For agents"
require_text README.md "SHIP_ONBOARD=1"
require_text README.md "manual dns"
require_text README.md "bootstrap-kind.sh"
require_text README.md "CLOUDFLARE_API_TOKEN"
require_text README.md "SHIP_DASHBOARD_SERVICE"
require_text README.md "Security"
require_text README.md "make test"
require_text deploy-system/README.md "./deploy-dashboard.sh"
require_text deploy-system/README.md "Default DNS mode"
require_text deploy-system/README.md "Cloudflare DNS mode"
require_text install.sh "SHIP_ONBOARD"
require_text install.sh "CLOUDFLARE_API_TOKEN"
require_text scripts/bootstrap-kind.sh "kind create cluster"
require_text scripts/bootstrap-kind.sh "envoyproxy/gateway-helm"
require_text scripts/bootstrap-kind.sh "tailscale/tailscale-operator"
require_text .env.example "SHIP_DOMAIN=mydomain.com"
require_text .env.example "CLOUDFLARE_API_TOKEN"
require_text CONTRIBUTING.md "Pull Request Checklist"
require_text SECURITY.md "Security Model"
require_text start-app/package.json "\"test\""
require_text start-app/package.json "\"typecheck\""
require_text start-app/package.json "\"lint\""
require_text start-app/Dockerfile "USER ship"
require_text .gitignore ".omo/"
require_text .gitignore "start-app/node_modules/"

printf 'release-readiness: ok\n'
