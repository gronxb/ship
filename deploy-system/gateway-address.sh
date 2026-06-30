#!/bin/sh
set -eu

. "$(dirname "$0")/ship-env.sh"

namespace="${1:-$SHIP_GATEWAY_NAMESPACE}"
name="${2:-$SHIP_GATEWAY_NAME}"

kubectl get gateway "$name" -n "$namespace" -o json | ruby -rjson -e '
  gateway = JSON.parse(STDIN.read)
  addresses = gateway.dig("status", "addresses") || []
  address = addresses.find { |item| item["value"].to_s != "" }
  abort("gateway has no status.addresses yet") unless address
  puts address.fetch("value")
'
