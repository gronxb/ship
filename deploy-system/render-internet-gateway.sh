#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

secret="wildcard-$(printf '%s' "$SHIP_DOMAIN" | tr . -)-tls"
export SHIP_TLS_SECRET="$secret"

ruby -e '
  input = STDIN.read
  domain = ENV.fetch("SHIP_DOMAIN")
  gateway_ns = ENV.fetch("SHIP_GATEWAY_NAMESPACE")
  internet_gateway = ENV.fetch("SHIP_INTERNET_GATEWAY_NAME")
  secret = ENV.fetch("SHIP_TLS_SECRET")

  output = input
    .gsub("ship-system", gateway_ns)
    .gsub("ship-internet", internet_gateway)
    .gsub("*.example.com", "*." + domain)
    .gsub("wildcard-example-com-tls", secret)

  print output
' < internet-gateway.yaml
