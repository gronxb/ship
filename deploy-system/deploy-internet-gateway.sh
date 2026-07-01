#!/bin/sh
set -eu

cd "$(dirname "$0")"
. ./ship-env.sh

if ! kubectl get ingressclass tailscale >/dev/null 2>&1; then
  cat <<EOF
missing: Tailscale IngressClass is not installed.
run: ./bootstrap-kind.sh
EOF
  exit 1
fi

printf 'ok: Tailscale IngressClass is installed\n\n'
cat <<EOF
manual action: enable Tailscale Funnel for the Kubernetes operator tag in the tailnet policy:

{
  "nodeAttrs": [
    {
      "target": ["tag:k8s"],
      "attr": ["funnel"]
    }
  ]
}

Then use the dashboard's "Expose to internet" button or:

ship --service <service> --exposure internet

The public URL appears in:

kubectl get ingress -n $SHIP_NAMESPACE
EOF
