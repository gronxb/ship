#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

case "${SHIP_DNS:-manual}" in
  manual)
    printf 'manual dns: skipped Cloudflare Tunnel connector\n'
    exit 0
    ;;
  auto | cloudflare)
    ;;
  *)
    printf 'SHIP_DNS must be auto, manual, or cloudflare\n' >&2
    exit 2
    ;;
esac

token="${CLOUDFLARE_API_TOKEN:-${CF_API_TOKEN:-}}"
if [ -z "$token" ]; then
  if [ "${SHIP_DNS:-manual}" = "cloudflare" ]; then
    printf 'missing CLOUDFLARE_API_TOKEN with Cloudflare Tunnel Edit\n' >&2
    exit 1
  fi
  printf 'automatic Cloudflare Tunnel skipped: missing CLOUDFLARE_API_TOKEN\n'
  exit 0
fi

tunnel_name="${SHIP_CLOUDFLARE_TUNNEL_NAME:-ship-$SHIP_DOMAIN}"
if [ "${SHIP_DRY_RUN:-0}" = "1" ]; then
  printf 'dry-run: ensure Cloudflare Tunnel %s and in-cluster cloudflared connector\n' "$tunnel_name"
  exit 0
fi

cloudflare_state="$(mktemp)"
cleanup_cloudflare_state() {
  rm -f "$cloudflare_state"
}
trap cleanup_cloudflare_state EXIT

ruby -rjson -rnet/http -ruri -rshellwords -e '
  API = "https://api.cloudflare.com/client/v4"

  def request(method, path, token, body = nil)
    uri = URI("#{API}#{path}")
    http = Net::HTTP.new(uri.host, uri.port)
    http.use_ssl = true
    req = Object.const_get("Net::HTTP::#{method}").new(uri)
    req["Authorization"] = "Bearer #{token}"
    req["Content-Type"] = "application/json"
    req.body = JSON.generate(body) if body
    res = http.request(req)
    json = JSON.parse(res.body)
    unless json["success"]
      message = json.fetch("errors", []).map { |err| err["message"] }.join("; ")
      hint = if path.include?("/cfd_tunnel")
        " Ensure CLOUDFLARE_API_TOKEN has Account Cloudflare Tunnel Edit for the account."
      elsif path.start_with?("/zones")
        " Ensure CLOUDFLARE_API_TOKEN has Zone Read, or set CLOUDFLARE_ACCOUNT_ID."
      else
        ""
      end
      abort("cloudflare api failed at #{method.upcase} #{path}: #{message.empty? ? res.body : message}#{hint}")
    end
    json.fetch("result")
  end

  token, account_id, zone_id, tunnel_name, domain = ARGV
  if account_id.empty? || zone_id.empty?
    zones = request("Get", "/zones?name=#{URI.encode_www_form_component(domain)}&per_page=50", token)
    zone = zones.find { |item| item["name"] == domain }
    if zone
      account_id = zone.dig("account", "id").to_s if account_id.empty?
      zone_id = zone.fetch("id") if zone_id.empty?
    end
  end

  if account_id.empty?
    accounts = request("Get", "/accounts?per_page=100", token) rescue []
    account_id = accounts.first.fetch("id") if accounts.length == 1
  end

  if account_id.empty?
    abort("missing CLOUDFLARE_ACCOUNT_ID; set it when the token cannot reveal the #{domain} account")
  end
  abort("missing CLOUDFLARE_ZONE_ID; set it when the token cannot reveal the #{domain} zone") if zone_id.empty?

  tunnels = request(
    "Get",
    "/accounts/#{URI.encode_www_form_component(account_id)}/cfd_tunnel?name=#{URI.encode_www_form_component(tunnel_name)}&is_deleted=false&per_page=100",
    token,
  )
  tunnel = tunnels.find { |item| item["name"] == tunnel_name }
  if tunnel
    tunnel_id = tunnel.fetch("id")
    tunnel_token = request("Get", "/accounts/#{account_id}/cfd_tunnel/#{tunnel_id}/token", token)
  else
    tunnel = request(
      "Post",
      "/accounts/#{account_id}/cfd_tunnel",
      token,
      { name: tunnel_name, config_src: "cloudflare" },
    )
    tunnel_id = tunnel.fetch("id")
    tunnel_token = tunnel.fetch("token")
  end

  config = request("Get", "/accounts/#{account_id}/cfd_tunnel/#{tunnel_id}/configurations", token) rescue nil
  ingress = config&.dig("config", "ingress")
  if ingress.nil? || ingress.empty?
    request(
      "Put",
      "/accounts/#{account_id}/cfd_tunnel/#{tunnel_id}/configurations",
      token,
      { config: { ingress: [{ service: "http_status:404" }] } },
    )
  end

  puts "CLOUDFLARE_ACCOUNT_ID=#{Shellwords.escape(account_id)}"
  puts "CLOUDFLARE_ZONE_ID=#{Shellwords.escape(zone_id)}"
  puts "CLOUDFLARE_TUNNEL_ID=#{Shellwords.escape(tunnel_id)}"
  puts "CLOUDFLARE_TUNNEL_TOKEN=#{Shellwords.escape(tunnel_token)}"
' "$token" "${CLOUDFLARE_ACCOUNT_ID:-${CF_ACCOUNT_ID:-}}" "${CLOUDFLARE_ZONE_ID:-${CF_ZONE_ID:-}}" "$tunnel_name" "$SHIP_DOMAIN" > "$cloudflare_state"

. "$cloudflare_state"

config="${SHIP_CONFIG:-${XDG_CONFIG_HOME:-$HOME/.config}/ship/config.env}"
mkdir -p "$(dirname "$config")"
touch "$config"
chmod 600 "$config"
persist_config() {
  key="$1"
  value="$2"
  tmp="$(mktemp)"
  grep -v "^$key=" "$config" > "$tmp" || true
  printf '%s=%s\n' "$key" "$value" >> "$tmp"
  cat "$tmp" > "$config"
  rm -f "$tmp"
}
persist_config CLOUDFLARE_ACCOUNT_ID "$CLOUDFLARE_ACCOUNT_ID"
persist_config CLOUDFLARE_ZONE_ID "$CLOUDFLARE_ZONE_ID"
persist_config CLOUDFLARE_TUNNEL_ID "$CLOUDFLARE_TUNNEL_ID"
persist_config SHIP_CLOUDFLARE_TUNNEL_NAME "$tunnel_name"

kubectl create namespace "$SHIP_GATEWAY_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret generic ship-cloudflared \
  -n "$SHIP_GATEWAY_NAMESPACE" \
  --from-literal=tunnel-token="$CLOUDFLARE_TUNNEL_TOKEN" \
  --dry-run=client \
  -o yaml | kubectl apply -f -

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ship-cloudflared
  namespace: $SHIP_GATEWAY_NAMESPACE
  labels:
    app.kubernetes.io/name: ship-cloudflared
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: ship-cloudflared
  template:
    metadata:
      labels:
        app.kubernetes.io/name: ship-cloudflared
    spec:
      containers:
        - name: cloudflared
          image: cloudflare/cloudflared:latest
          args:
            - tunnel
            - --no-autoupdate
            - run
            - --token
            - \$(TUNNEL_TOKEN)
          env:
            - name: TUNNEL_TOKEN
              valueFrom:
                secretKeyRef:
                  name: ship-cloudflared
                  key: tunnel-token
EOF

kubectl rollout status deployment/ship-cloudflared -n "$SHIP_GATEWAY_NAMESPACE" --timeout=180s
printf 'ok: Cloudflare Tunnel %s is running for public internet exposure\n' "$CLOUDFLARE_TUNNEL_ID"
