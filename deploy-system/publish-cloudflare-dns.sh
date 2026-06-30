#!/bin/sh
set -eu

. "$(dirname "$0")/ship-env.sh"

domain="${DOMAIN:-$SHIP_DOMAIN}"
record_name="${RECORD_NAME:-*.$SHIP_DOMAIN}"
target="${1:-}"
zone_id="${CLOUDFLARE_ZONE_ID:-${CF_ZONE_ID:-}}"

if [ -z "$target" ]; then
  target="$(./gateway-address.sh)"
fi

if [ -z "${CLOUDFLARE_API_TOKEN:-${CF_API_TOKEN:-}}" ]; then
  wrangler="${WRANGLER:-}"
  if [ -z "$wrangler" ] && command -v wrangler >/dev/null 2>&1; then
    wrangler="$(command -v wrangler)"
  fi
  if [ -n "$wrangler" ]; then
    "$wrangler" whoami >/dev/null 2>&1 || true
  fi
fi

ruby -rjson -rnet/http -ruri -ripaddr -rtime -e '
  API = "https://api.cloudflare.com/client/v4"
  LOGIN = "./cloudflare-login.sh"

  def wrangler_config_path
    return ENV["WRANGLER_AUTH_CONFIG"] if ENV["WRANGLER_AUTH_CONFIG"].to_s != ""

    [
      File.expand_path("~/Library/Preferences/.wrangler/config/default.toml"),
      File.expand_path("~/.wrangler/config/default.toml"),
      File.expand_path("#{ENV["XDG_CONFIG_HOME"]}/.wrangler/config/default.toml"),
    ].find { |path| path && File.file?(path) }
  end

  def toml_string(source, key)
    source[/^#{Regexp.escape(key)}\s*=\s*"([^"]+)"/, 1]
  end

  def toml_array(source, key)
    raw = source[/^#{Regexp.escape(key)}\s*=\s*\[(.*?)\]/m, 1]
    return [] unless raw

    raw.scan(/"([^"]+)"/).flatten
  end

  def auth_token
    env_token = ENV["CLOUDFLARE_API_TOKEN"].to_s
    env_token = ENV["CF_API_TOKEN"].to_s if env_token.empty?
    return env_token unless env_token.empty?

    path = wrangler_config_path
    abort("missing Wrangler login. Run #{LOGIN}, then retry.") unless path

    config = File.read(path)
    token = toml_string(config, "oauth_token") || toml_string(config, "api_token")
    abort("missing Wrangler OAuth token. Run #{LOGIN}, then retry.") unless token

    scopes = toml_array(config, "scopes")
    unless scopes.include?("zone:read")
      abort("Wrangler login is missing zone:read. Run #{LOGIN}, then retry.")
    end

    expiry = toml_string(config, "expiration_time")
    if expiry && Time.parse(expiry) <= Time.now + 300
      abort("Wrangler OAuth token is expired or near expiry. Run #{LOGIN}, then retry.")
    end

    token
  end

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
      hint = if res.code.to_i == 403
        " Run #{LOGIN}; if Cloudflare still rejects DNS writes, use CLOUDFLARE_API_TOKEN with Zone DNS Edit."
      else
        ""
      end
      abort("cloudflare api failed: #{message.empty? ? res.body : message}#{hint}")
    end
    json
  end

  zone_id, domain, record_name, target = ARGV
  token = auth_token

  if zone_id.empty?
    zone = request("Get", "/zones?name=#{URI.encode_www_form_component(domain)}", token)
      .fetch("result")
      .find { |item| item["name"] == domain }
    abort("cloudflare zone not found: #{domain}") unless zone
    zone_id = zone.fetch("id")
  end

  record_type =
    begin
      IPAddr.new(target)
      target.include?(":") ? "AAAA" : "A"
    rescue IPAddr::InvalidAddressError
      target = target.sub(/\.\z/, "")
      "CNAME"
    end

  existing = request(
    "Get",
    "/zones/#{zone_id}/dns_records?name=#{URI.encode_www_form_component(record_name)}&per_page=100",
    token,
  ).fetch("result")

  existing.each do |record|
    next if record["type"] == record_type
    request("Delete", "/zones/#{zone_id}/dns_records/#{record.fetch("id")}", token)
  end

  body = {
    type: record_type,
    name: record_name,
    content: target,
    ttl: 60,
    proxied: false,
    comment: "Ship Kubernetes Gateway. DNS-only; do not proxy.",
  }

  current = existing.find { |record| record["type"] == record_type }
  if current
    request("Patch", "/zones/#{zone_id}/dns_records/#{current.fetch("id")}", token, body)
    action = "updated"
  else
    request("Post", "/zones/#{zone_id}/dns_records", token, body)
    action = "created"
  end

  puts "#{action}: #{record_name} #{record_type} #{target} proxied=false"
' "$zone_id" "$domain" "$record_name" "$target"
