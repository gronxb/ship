import { z } from "zod"

export type Deployment = {
  readonly serviceName: string
  readonly host: string
  readonly image: string
  readonly namespace: string
  readonly port: number
  readonly exposure: Exposure
  readonly tailscaleOnly: boolean
  readonly dryRun: boolean
  readonly commands: readonly string[]
  readonly manifest: string
  readonly containerLogs: string
  readonly createdAt?: string
}

export type Exposure = "tailscale" | "internet"

export type DeploymentsResponse = {
  readonly deployments: readonly Deployment[]
}

export const DeploymentsResponse = z.object({
  deployments: z.array(
    z.object({
      serviceName: z.string(),
      host: z.string(),
      image: z.string(),
      namespace: z.string(),
      port: z.number(),
      exposure: z.enum(["tailscale", "internet"]).default("tailscale"),
      tailscaleOnly: z.boolean(),
      dryRun: z.boolean(),
      commands: z.array(z.string()),
      manifest: z.string(),
      containerLogs: z.string().default(""),
      createdAt: z.string().optional(),
    })
  ),
})
