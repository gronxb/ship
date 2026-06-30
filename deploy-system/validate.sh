#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

tmp="$(mktemp)"
cleanup() {
  rm -f "$tmp"
}
trap cleanup EXIT

sync_output="$(./sync-host.sh)"
./render.sh > "$tmp"

ruby -ryaml -e '
  docs = YAML.load_stream(File.read(ARGV.fetch(0)))
  tailscale_ip, dashboard_port = ARGV.fetch(1), ARGV.fetch(2).to_i
  domain = ARGV.fetch(3)
  dashboard_host = ARGV.fetch(4)
  abort("missing Gateway") unless docs.any? { |doc| doc["kind"] == "Gateway" }
  abort("missing wildcard hostname") unless docs.any? { |doc|
    doc.dig("spec", "listeners")&.any? { |listener| listener["hostname"] == "*." + domain }
  }
  abort("missing tailscale load balancer") unless docs.any? { |doc|
    doc.dig("spec", "provider", "kubernetes", "envoyService", "loadBalancerClass") == "tailscale"
  }
  proxy = docs.find { |doc| doc["kind"] == "ConfigMap" && doc.dig("metadata", "name") == "ship-dashboard-proxy-caddy" }
  abort("missing proxy config") unless proxy
  config = proxy.dig("data", "Caddyfile").to_s
  abort("missing deploy dashboard host matcher") unless config.include?("@deploy_dashboard host " + dashboard_host)
  abort("missing deploy dashboard upstream") unless config.include?("reverse_proxy #{tailscale_ip}:#{dashboard_port}")
  abort("proxy must not wildcard-forward deployed services to the Mac") if config.include?("handle {\n    \t\t\treverse_proxy")
  abort("missing forwarded host override") unless config.include?("header_up X-Forwarded-Host {host}")
  abort("missing forwarded proto override") unless config.include?("header_up X-Forwarded-Proto https")
  services = docs.select { |doc| doc["kind"] == "Service" }.map { |doc| doc.dig("metadata", "name") }
  abort("missing proxy service") unless services.include?("ship-dashboard-proxy")
  abort("missing caddy proxy deployment") unless docs.any? { |doc|
    doc["kind"] == "Deployment" &&
      doc.dig("metadata", "name") == "ship-dashboard-proxy" &&
      doc.dig("spec", "template", "spec", "containers", 0, "image").to_s.start_with?("caddy:")
  }
  routes = docs.select { |doc| doc["kind"] == "HTTPRoute" }
  abort("missing k8s deploy dashboard exact host route") unless routes.any? { |route|
    route.dig("metadata", "name") == "k8s-deploy-dashboard" &&
      route.dig("spec", "hostnames") == [dashboard_host]
  }
  abort("routes must target proxy service") unless routes.all? { |route|
    route.dig("spec", "rules", 0, "backendRefs", 0, "name") == "ship-dashboard-proxy"
  }
' "$tmp" "$(tailscale ip -4 | sed -n '1p')" "$SHIP_DASHBOARD_PORT" "$SHIP_DOMAIN" "$SHIP_DASHBOARD_HOST"

printf '%s\n' "$sync_output"
printf 'ok: rendered ship tailnet-only gateway config\n'
