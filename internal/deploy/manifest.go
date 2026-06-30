package deploy

import (
	"bytes"
	"fmt"
)

func manifestFor(opts Options, host string, image string) string {
	var b bytes.Buffer
	tailscaleOnly := opts.Exposure == "tailscale"
	tailscaleOnlyLabel := "false"
	if tailscaleOnly {
		tailscaleOnlyLabel = "true"
	}
	envFrom := ""
	if opts.EnvFile != "" {
		envFrom = fmt.Sprintf(`          envFrom:
            - secretRef:
                name: %s-env
`, opts.ServiceName)
	}
	fmt.Fprintf(&b, `apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    gateway-access: allowed
    ship.local/exposure: "%s"
    ship.local/tailscale-only: "%s"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    ship.local/exposure: "%s"
    ship.local/tailscale-only: "%s"
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: %s
  template:
    metadata:
      labels:
        app.kubernetes.io/name: %s
    spec:
      containers:
        - name: app
          image: %s
          imagePullPolicy: IfNotPresent
%s
          ports:
            - name: http
              containerPort: %d
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    ship.local/exposure: "%s"
    ship.local/tailscale-only: "%s"
spec:
  selector:
    app.kubernetes.io/name: %s
  ports:
    - name: http
      port: 80
      targetPort: http
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    ship.local/exposure: "%s"
    ship.local/tailscale-only: "%s"
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
`, opts.Namespace, opts.Exposure, tailscaleOnlyLabel, opts.ServiceName, opts.Namespace, opts.ServiceName, opts.Exposure, tailscaleOnlyLabel, opts.ServiceName, opts.ServiceName, image, envFrom, opts.Port, opts.ServiceName, opts.Namespace, opts.ServiceName, opts.Exposure, tailscaleOnlyLabel, opts.ServiceName, opts.ServiceName, opts.Namespace, opts.ServiceName, opts.Exposure, tailscaleOnlyLabel, opts.GatewayName, opts.GatewayNamespace, host, opts.ServiceName)
	return b.String()
}
