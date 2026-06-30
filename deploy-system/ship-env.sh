#!/bin/sh
set -eu

config="${SHIP_CONFIG:-${XDG_CONFIG_HOME:-$HOME/.config}/ship/config.env}"

if [ -f "../.env" ]; then
  . "../.env"
fi

if [ -f "$config" ]; then
  . "$config"
fi

: "${SHIP_DOMAIN:=example.com}"
: "${SHIP_GATEWAY_NAMESPACE:=ship-system}"
: "${SHIP_GATEWAY_NAME:=ship-tailscale}"
: "${SHIP_INTERNET_GATEWAY_NAME:=ship-internet}"
: "${SHIP_DASHBOARD_HOST:=k8s.$SHIP_DOMAIN}"
: "${SHIP_DASHBOARD_PORT:=9292}"

export SHIP_DOMAIN
export SHIP_GATEWAY_NAMESPACE
export SHIP_GATEWAY_NAME
export SHIP_INTERNET_GATEWAY_NAME
export SHIP_DASHBOARD_HOST
export SHIP_DASHBOARD_PORT
