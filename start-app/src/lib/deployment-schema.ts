import { z } from "zod"

export const Exposure = z.enum(["tailscale", "internet"])

export const DeploymentResult = z.object({
  serviceName: z.string(),
  host: z.string(),
  image: z.string(),
  namespace: z.string(),
  dockerfilePath: z.string(),
  contextDir: z.string(),
  envFilePath: z.string().optional(),
  port: z.number(),
  exposure: Exposure.default("tailscale"),
  tailscaleOnly: z.boolean(),
  dryRun: z.boolean(),
  commands: z.array(z.string()),
  manifest: z.string(),
  containerLogs: z.string().default(""),
  createdAt: z.string().optional(),
})

export const HttpRouteList = z.object({
  items: z.array(
    z.object({
      metadata: z.object({
        name: z.string(),
        namespace: z.string().default("ship-services"),
        labels: z.record(z.string(), z.string()).optional(),
      }),
      spec: z.object({
        hostnames: z.array(z.string()).default([]),
        parentRefs: z
          .array(
            z.object({
              name: z.string(),
            })
          )
          .default([]),
      }),
    })
  ),
})

export type Exposure = z.infer<typeof Exposure>
export type DeploymentResult = z.infer<typeof DeploymentResult>
export type HttpRoute = z.infer<typeof HttpRouteList>["items"][number]
