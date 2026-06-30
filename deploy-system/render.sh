#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

./sync-host.sh >/dev/null
tailscale_ip="$(tailscale ip -4 | sed -n '1p')"

secret="wildcard-$(printf '%s' "$SHIP_DOMAIN" | tr . -)-tls"
export SHIP_TLS_SECRET="$secret"
export SHIP_DASHBOARD_UPSTREAM="${tailscale_ip}:${SHIP_DASHBOARD_PORT}"

kubectl kustomize . | ruby -e '
  input = STDIN.read
  domain = ENV.fetch("SHIP_DOMAIN")
  dashboard = ENV.fetch("SHIP_DASHBOARD_HOST")
  gateway_ns = ENV.fetch("SHIP_GATEWAY_NAMESPACE")
  tailscale_gateway = ENV.fetch("SHIP_GATEWAY_NAME")
  internet_gateway = ENV.fetch("SHIP_INTERNET_GATEWAY_NAME")
  secret = ENV.fetch("SHIP_TLS_SECRET")
  upstream = ENV.fetch("SHIP_DASHBOARD_UPSTREAM")

  output = input
    .gsub("ship-system", gateway_ns)
    .gsub("ship-tailscale", tailscale_gateway)
    .gsub("ship-internet", internet_gateway)
    .gsub("*.example.com", "*." + domain)
    .gsub("k8s.example.com", dashboard)
    .gsub("wildcard-example-com-tls", secret)
    .gsub("127.0.0.1:9292", upstream)

  print output
'
