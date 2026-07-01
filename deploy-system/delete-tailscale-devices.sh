#!/bin/sh
set -eu

. "$(dirname "$0")/ship-env.sh"

client_id="${TAILSCALE_CLIENT_ID:-${TAILSCALE_OAUTH_CLIENT_ID:-${TS_OAUTH_CLIENT_ID:-}}}"
client_secret="${TAILSCALE_CLIENT_SECRET:-${TAILSCALE_OAUTH_CLIENT_SECRET:-${TS_OAUTH_CLIENT_SECRET:-}}}"
tailnet="${TAILSCALE_TAILNET:-${SHIP_TAILNET:-${TAILNET:--}}}"

if [ -z "$client_id" ] || [ -z "$client_secret" ]; then
  printf 'skipped: missing Tailscale OAuth credentials for device cleanup\n'
  exit 0
fi

if [ "${SHIP_DRY_RUN:-0}" = "1" ]; then
  printf 'dry-run: delete Tailscale devices named ship-tailscale* and tailscale-operator*\n'
  exit 0
fi

TAILSCALE_CLIENT_ID="$client_id" \
TAILSCALE_CLIENT_SECRET="$client_secret" \
TAILSCALE_TAILNET="$tailnet" \
ruby -rjson -rnet/http -ruri -e '
  API = "https://api.tailscale.com/api/v2"
  NAMES = ["ship-tailscale", "tailscale-operator"].freeze
  TAGS = ["tag:k8s", "tag:k8s-operator"].freeze

  def request(method, path, token = nil, body = nil)
    uri = URI("#{API}#{path}")
    http = Net::HTTP.new(uri.host, uri.port)
    http.use_ssl = true
    req = Object.const_get("Net::HTTP::#{method}").new(uri)
    req["Authorization"] = "Bearer #{token}" if token
    req["Content-Type"] = "application/x-www-form-urlencoded"
    req.body = body if body
    res = http.request(req)
    abort("tailscale api failed: #{res.code} #{res.body}") unless res.code.to_i.between?(200, 299)
    res.body.empty? ? {} : JSON.parse(res.body)
  end

  form = URI.encode_www_form(
    "client_id" => ENV.fetch("TAILSCALE_CLIENT_ID"),
    "client_secret" => ENV.fetch("TAILSCALE_CLIENT_SECRET"),
    "scope" => "devices:core",
  )
  token = request("Post", "/oauth/token", nil, form).fetch("access_token")
  tailnet = URI.encode_www_form_component(ENV.fetch("TAILSCALE_TAILNET"))
  devices = request("Get", "/tailnet/#{tailnet}/devices", token).fetch("devices")
  targets = devices.select do |device|
    name = device.fetch("name", "")
    tags = Array(device["tags"])
    NAMES.any? { |prefix| name.start_with?(prefix) } && TAGS.any? { |tag| tags.include?(tag) }
  end
  targets.each do |device|
    request("Delete", "/device/#{device.fetch("id")}", token)
    puts "deleted: #{device.fetch("name")}"
  end
  puts "deleted: Tailscale Ship devices (#{targets.length} devices)"
'
