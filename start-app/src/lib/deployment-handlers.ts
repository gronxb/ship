import { z } from "zod"

import { gatewayNamespace, internetGatewayName } from "./deployment-config"
import { execFileAsync } from "./deployment-command"
import { readMergedDeployments } from "./deployment-state"

const exposureUpdate = z.object({
  namespace: z
    .string()
    .regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/)
    .optional(),
  serviceName: z.string().regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/),
  exposure: z.literal("internet"),
})

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

  await execFileAsync("kubectl", [
    "patch",
    "httproute",
    serviceName,
    "-n",
    deployment.namespace,
    "--type=merge",
    "-p",
    JSON.stringify({
      metadata: {
        labels: {
          "ship.local/exposure": "internet",
          "ship.local/tailscale-only": "false",
        },
      },
      spec: {
        parentRefs: [
          {
            group: "gateway.networking.k8s.io",
            kind: "Gateway",
            name: internetGatewayName,
            namespace: gatewayNamespace,
          },
        ],
      },
    }),
  ])

  return Response.json({
    deployment: {
      serviceName,
      exposure: "internet",
      tailscaleOnly: false,
    },
  })
}
