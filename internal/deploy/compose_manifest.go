package deploy

import (
	"bytes"
	"fmt"
)

func composeManifestFor(opts Options, host string, hostGateway string, publishedPort int) string {
	var b bytes.Buffer
	tailscaleOnly := opts.Exposure == "tailscale"
	fmt.Fprintf(&b, `apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    gateway-access: allowed
    ship.local/exposure: "%s"
    ship.local/tailscale-only: "%t"
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    ship.local/runtime: compose
    ship.local/exposure: "%s"
    ship.local/tailscale-only: "%t"
spec:
  ports:
    - name: http
      port: 80
      targetPort: http
---
apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: %s-compose
  namespace: %s
  labels:
    kubernetes.io/service-name: %s
    app.kubernetes.io/name: %s
    ship.local/runtime: compose
addressType: IPv4
ports:
  - name: http
    protocol: TCP
    port: %d
endpoints:
  - addresses:
      - "%s"
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    ship.local/runtime: compose
    ship.local/exposure: "%s"
    ship.local/tailscale-only: "%t"
spec:
  parentRefs:
    - group: gateway.networking.k8s.io
      kind: Gateway
      name: %s
      namespace: %s
  hostnames:
    - %s
  rules:
    - backendRefs:
        - group: ""
          kind: Service
          name: %s
          port: 80
`, opts.Namespace, opts.Exposure, tailscaleOnly, opts.ServiceName, opts.Namespace, opts.ServiceName, opts.Exposure, tailscaleOnly, opts.ServiceName, opts.Namespace, opts.ServiceName, opts.ServiceName, publishedPort, hostGateway, opts.ServiceName, opts.Namespace, opts.ServiceName, opts.Exposure, tailscaleOnly, opts.GatewayName, opts.GatewayNamespace, host, opts.ServiceName)
	return b.String()
}
