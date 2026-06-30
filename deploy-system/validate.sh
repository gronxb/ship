#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

tmp="$(mktemp)"
cleanup() {
  rm -f "$tmp"
}
trap cleanup EXIT

./render.sh > "$tmp"

ruby -ryaml -e '
  docs = YAML.load_stream(File.read(ARGV.fetch(0)))
  domain = ARGV.fetch(1)
  abort("missing Gateway") unless docs.any? { |doc| doc["kind"] == "Gateway" }
  abort("missing wildcard hostname") unless docs.any? { |doc|
    doc.dig("spec", "listeners")&.any? { |listener| listener["hostname"] == "*." + domain }
  }
  abort("missing tailscale load balancer") unless docs.any? { |doc|
    doc.dig("spec", "provider", "kubernetes", "envoyService", "loadBalancerClass") == "tailscale"
  }
  forbidden = [
    ["Deployment", "ship-dashboard-proxy"],
    ["Service", "ship-dashboard-proxy"],
    ["ConfigMap", "ship-dashboard-proxy-caddy"],
    ["HTTPRoute", "k8s-deploy-dashboard"],
  ]
  forbidden.each do |kind, name|
    abort("default gateway render must not include #{kind}/#{name}") if docs.any? { |doc|
      doc["kind"] == kind && doc.dig("metadata", "name") == name
    }
  end
' "$tmp" "$SHIP_DOMAIN"

printf 'ok: rendered ship tailnet-only gateway config\n'
