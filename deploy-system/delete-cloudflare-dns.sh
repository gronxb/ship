#!/bin/sh
set -eu

. "$(dirname "$0")/ship-env.sh"

domain="${DOMAIN:-$SHIP_DOMAIN}"
record_name="${RECORD_NAME:-*.$SHIP_DOMAIN}"
zone_id="${CLOUDFLARE_ZONE_ID:-${CF_ZONE_ID:-}}"

if [ "${SHIP_DRY_RUN:-0}" = "1" ]; then
  printf 'dry-run: delete %s from Cloudflare DNS\n' "$record_name"
  exit 0
fi

ruby -rjson -rnet/http -ruri -e '
  API = "https://api.cloudflare.com/client/v4"

  def auth_token
    env_token = ENV["CLOUDFLARE_API_TOKEN"].to_s
    env_token = ENV["CF_API_TOKEN"].to_s if env_token.empty?
    abort("missing CLOUDFLARE_API_TOKEN") if env_token.empty?
    env_token
  end

  def request(method, path, token)
    uri = URI("#{API}#{path}")
    http = Net::HTTP.new(uri.host, uri.port)
    http.use_ssl = true
    req = Object.const_get("Net::HTTP::#{method}").new(uri)
    req["Authorization"] = "Bearer #{token}"
    req["Content-Type"] = "application/json"
    res = http.request(req)
    json = JSON.parse(res.body)
    abort("cloudflare api failed: #{res.body}") unless json["success"]
    json
  end

  zone_id, domain, record_name = ARGV
  token = auth_token
  if zone_id.empty?
    zone = request("Get", "/zones?name=#{URI.encode_www_form_component(domain)}", token)
      .fetch("result")
      .find { |item| item["name"] == domain }
    abort("cloudflare zone not found: #{domain}") unless zone
    zone_id = zone.fetch("id")
  end

  records = request(
    "Get",
    "/zones/#{zone_id}/dns_records?name=#{URI.encode_www_form_component(record_name)}&per_page=100",
    token,
  ).fetch("result")
  records.each do |record|
    request("Delete", "/zones/#{zone_id}/dns_records/#{record.fetch("id")}", token)
  end
  puts "deleted: #{record_name} (#{records.length} records)"
' "$zone_id" "$domain" "$record_name"
