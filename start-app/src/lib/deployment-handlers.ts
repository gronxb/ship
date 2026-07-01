import { z } from "zod"

import { execFileWithInput } from "./deployment-command"
import { readMergedDeployments } from "./deployment-state"

const exposureUpdate = z.object({
  namespace: z
    .string()
    .regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/)
    .optional(),
  serviceName: z.string().regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/),
  exposure: z.literal("internet"),
})

const tailscaleFunnelSetupCommand =
  "cd deploy-system && ./deploy-internet-gateway.sh"

export async function listDeployments(): Promise<Response> {
  return Response.json({ deployments: await readMergedDeployments() })
}

export async function changeDeploymentExposure(
  request: Request
): Promise<Response> {
  let body: unknown
  try {
    body = await request.json()
  } catch (caught) {
    if (caught instanceof SyntaxError) {
      return Response.json(
        { error: "invalid exposure update" },
        { status: 400 }
      )
    }
    throw caught
  }

  const parsed = exposureUpdate.safeParse(body)
  if (!parsed.success) {
    return Response.json({ error: "invalid exposure update" }, { status: 400 })
  }

  const { namespace: requestedNamespace, serviceName } = parsed.data
  const deployment = (await readMergedDeployments()).find((item) => {
    return (
      item.serviceName === serviceName &&
      (requestedNamespace === undefined ||
        item.namespace === requestedNamespace)
    )
  })
  if (!deployment) {
    return Response.json({ error: "deployment not found" }, { status: 404 })
  }
  if (!deployment.tailscaleOnly) {
    return Response.json({
      deployment: {
        serviceName,
        exposure: "internet",
        tailscaleOnly: false,
      },
    })
  }

  try {
    await execFileWithInput(
      "kubectl",
      ["apply", "-f", "-"],
      funnelIngressManifest(serviceName, deployment.namespace)
    )
  } catch (caught) {
    if (caught instanceof Error) {
      return Response.json(
        {
          error: `tailscale funnel not ready; run: ${tailscaleFunnelSetupCommand}`,
        },
        { status: 409 }
      )
    }
    throw caught
  }

  return Response.json({
    deployment: {
      serviceName,
      exposure: "internet",
      tailscaleOnly: false,
    },
  })
}

function funnelIngressManifest(serviceName: string, namespace: string): string {
  return `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ${serviceName}
  namespace: ${namespace}
  labels:
    app.kubernetes.io/name: ${serviceName}
    ship.local/exposure: "internet"
    ship.local/tailscale-only: "false"
  annotations:
    tailscale.com/funnel: "true"
spec:
  ingressClassName: tailscale
  defaultBackend:
    service:
      name: ${serviceName}
      port:
        number: 80
  tls:
    - hosts:
        - ${serviceName}
`
}
