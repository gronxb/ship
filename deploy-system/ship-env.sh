#!/bin/sh
set -eu

config="${SHIP_CONFIG:-${XDG_CONFIG_HOME:-$HOME/.config}/ship/config.env}"

env_ship_domain_set=${SHIP_DOMAIN+x}
env_ship_domain=${SHIP_DOMAIN-}
env_ship_namespace_set=${SHIP_NAMESPACE+x}
env_ship_namespace=${SHIP_NAMESPACE-}
env_ship_gateway_namespace_set=${SHIP_GATEWAY_NAMESPACE+x}
env_ship_gateway_namespace=${SHIP_GATEWAY_NAMESPACE-}
env_ship_gateway_name_set=${SHIP_GATEWAY_NAME+x}
env_ship_gateway_name=${SHIP_GATEWAY_NAME-}
env_ship_internet_gateway_name_set=${SHIP_INTERNET_GATEWAY_NAME+x}
env_ship_internet_gateway_name=${SHIP_INTERNET_GATEWAY_NAME-}
env_ship_dns_set=${SHIP_DNS+x}
env_ship_dns=${SHIP_DNS-}
env_ship_dashboard_host_set=${SHIP_DASHBOARD_HOST+x}
env_ship_dashboard_host=${SHIP_DASHBOARD_HOST-}
env_ship_image_prefix_set=${SHIP_IMAGE_PREFIX+x}
env_ship_image_prefix=${SHIP_IMAGE_PREFIX-}
env_kind_cluster_set=${KIND_CLUSTER+x}
env_kind_cluster=${KIND_CLUSTER-}
env_registry_set=${REGISTRY+x}
env_registry=${REGISTRY-}
env_ship_exposure_set=${SHIP_EXPOSURE+x}
env_ship_exposure=${SHIP_EXPOSURE-}
env_image_tag_set=${IMAGE_TAG+x}
env_image_tag=${IMAGE_TAG-}
env_ship_bin_set=${SHIP_BIN+x}
env_ship_bin=${SHIP_BIN-}
env_ship_dashboard_service_set=${SHIP_DASHBOARD_SERVICE+x}
env_ship_dashboard_service=${SHIP_DASHBOARD_SERVICE-}
env_ship_dashboard_service_account_set=${SHIP_DASHBOARD_SERVICE_ACCOUNT+x}
env_ship_dashboard_service_account=${SHIP_DASHBOARD_SERVICE_ACCOUNT-}
env_ship_acme_email_set=${SHIP_ACME_EMAIL+x}
env_ship_acme_email=${SHIP_ACME_EMAIL-}
env_cloudflare_api_token_set=${CLOUDFLARE_API_TOKEN+x}
env_cloudflare_api_token=${CLOUDFLARE_API_TOKEN-}
env_cf_api_token_set=${CF_API_TOKEN+x}
env_cf_api_token=${CF_API_TOKEN-}
env_cloudflare_zone_id_set=${CLOUDFLARE_ZONE_ID+x}
env_cloudflare_zone_id=${CLOUDFLARE_ZONE_ID-}
env_cf_zone_id_set=${CF_ZONE_ID+x}
env_cf_zone_id=${CF_ZONE_ID-}

if [ -f "../.env" ]; then
  . "../.env"
fi

if [ -f "$config" ]; then
  . "$config"
fi

if [ -n "$env_ship_domain_set" ]; then
  SHIP_DOMAIN=$env_ship_domain
fi
if [ -n "$env_ship_namespace_set" ]; then
  SHIP_NAMESPACE=$env_ship_namespace
fi
if [ -n "$env_ship_gateway_namespace_set" ]; then
  SHIP_GATEWAY_NAMESPACE=$env_ship_gateway_namespace
fi
if [ -n "$env_ship_gateway_name_set" ]; then
  SHIP_GATEWAY_NAME=$env_ship_gateway_name
fi
if [ -n "$env_ship_internet_gateway_name_set" ]; then
  SHIP_INTERNET_GATEWAY_NAME=$env_ship_internet_gateway_name
fi
if [ -n "$env_ship_dns_set" ]; then
  SHIP_DNS=$env_ship_dns
fi
if [ -n "$env_ship_dashboard_host_set" ]; then
  SHIP_DASHBOARD_HOST=$env_ship_dashboard_host
fi
if [ -n "$env_ship_image_prefix_set" ]; then
  SHIP_IMAGE_PREFIX=$env_ship_image_prefix
fi
if [ -n "$env_kind_cluster_set" ]; then
  KIND_CLUSTER=$env_kind_cluster
fi
if [ -n "$env_registry_set" ]; then
  REGISTRY=$env_registry
fi
if [ -n "$env_ship_exposure_set" ]; then
  SHIP_EXPOSURE=$env_ship_exposure
fi
if [ -n "$env_image_tag_set" ]; then
  IMAGE_TAG=$env_image_tag
fi
if [ -n "$env_ship_bin_set" ]; then
  SHIP_BIN=$env_ship_bin
fi
if [ -n "$env_ship_dashboard_service_set" ]; then
  SHIP_DASHBOARD_SERVICE=$env_ship_dashboard_service
fi
if [ -n "$env_ship_dashboard_service_account_set" ]; then
  SHIP_DASHBOARD_SERVICE_ACCOUNT=$env_ship_dashboard_service_account
fi
if [ -n "$env_ship_acme_email_set" ]; then
  SHIP_ACME_EMAIL=$env_ship_acme_email
fi
if [ -n "$env_cloudflare_api_token_set" ]; then
  CLOUDFLARE_API_TOKEN=$env_cloudflare_api_token
fi
if [ -n "$env_cf_api_token_set" ]; then
  CF_API_TOKEN=$env_cf_api_token
fi
if [ -n "$env_cloudflare_zone_id_set" ]; then
  CLOUDFLARE_ZONE_ID=$env_cloudflare_zone_id
fi
if [ -n "$env_cf_zone_id_set" ]; then
  CF_ZONE_ID=$env_cf_zone_id
fi

: "${SHIP_DOMAIN:=example.com}"
: "${SHIP_NAMESPACE:=ship-services}"
: "${SHIP_GATEWAY_NAMESPACE:=ship-system}"
: "${SHIP_GATEWAY_NAME:=ship-tailscale}"
: "${SHIP_INTERNET_GATEWAY_NAME:=ship-internet}"
: "${SHIP_DNS:=manual}"
: "${SHIP_IMAGE_PREFIX:=ship}"
: "${KIND_CLUSTER:=ship}"
: "${REGISTRY:=}"

export SHIP_DOMAIN
export SHIP_NAMESPACE
export SHIP_GATEWAY_NAMESPACE
export SHIP_GATEWAY_NAME
export SHIP_INTERNET_GATEWAY_NAME
export SHIP_DNS
export SHIP_IMAGE_PREFIX
export KIND_CLUSTER
export REGISTRY
if [ -n "${SHIP_DASHBOARD_HOST:-}" ]; then
  export SHIP_DASHBOARD_HOST
fi
if [ -n "${SHIP_EXPOSURE:-}" ]; then
  export SHIP_EXPOSURE
fi
if [ -n "${IMAGE_TAG:-}" ]; then
  export IMAGE_TAG
fi
if [ -n "${SHIP_BIN:-}" ]; then
  export SHIP_BIN
fi
if [ -n "${SHIP_DASHBOARD_SERVICE:-}" ]; then
  export SHIP_DASHBOARD_SERVICE
fi
if [ -n "${SHIP_DASHBOARD_SERVICE_ACCOUNT:-}" ]; then
  export SHIP_DASHBOARD_SERVICE_ACCOUNT
fi
if [ -n "${SHIP_ACME_EMAIL:-}" ]; then
  export SHIP_ACME_EMAIL
fi
if [ -n "${CLOUDFLARE_API_TOKEN:-}" ]; then
  export CLOUDFLARE_API_TOKEN
fi
if [ -n "${CF_API_TOKEN:-}" ]; then
  export CF_API_TOKEN
fi
if [ -n "${CLOUDFLARE_ZONE_ID:-}" ]; then
  export CLOUDFLARE_ZONE_ID
fi
if [ -n "${CF_ZONE_ID:-}" ]; then
  export CF_ZONE_ID
fi
