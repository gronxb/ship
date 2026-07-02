import { z } from "zod"

const cloudflareEnvelopeError = z.object({ message: z.string().default("") })
const cloudflareRecord = z.object({ id: z.string(), type: z.string() })
const cloudflareIngressRule = z.object({
  hostname: z.string().optional(),
  service: z.string(),
})
const cloudflareTunnelConfiguration = z.object({
  config: z
    .object({
      ingress: z.array(cloudflareIngressRule).default([]),
    })
    .default({ ingress: [] }),
})

type CloudflareRecord = z.infer<typeof cloudflareRecord>
type CloudflareIngressRule = z.infer<typeof cloudflareIngressRule>
type CloudflareTunnelConfiguration = z.infer<
  typeof cloudflareTunnelConfiguration
>
type DNSRecordBody = {
  readonly type: string
  readonly name: string
  readonly content: string
  readonly ttl: number
  readonly proxied: boolean
  readonly comment: string
}

type CloudflareConfig = {
  readonly apiEndpoint: string
  readonly apiToken: string
  readonly accountID: string
  readonly tunnelID: string
  readonly zoneID: string
}

type CloudflareTunnelRoute = {
  readonly host: string
  readonly namespace: string
  readonly serviceName: string
}

export class CloudflareTunnelSetupError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "CloudflareTunnelSetupError"
  }
}

export async function exposeCloudflareTunnelRoute(
  route: CloudflareTunnelRoute
): Promise<void> {
  const config = cloudflareConfig()
  const current = await tunnelConfiguration(config)
  await putTunnelConfiguration(config, {
    config: {
      ingress: mergeIngressRules(current.config.ingress, {
        hostname: route.host,
        service: `http://${route.serviceName}.${route.namespace}.svc.cluster.local:80`,
      }),
    },
  })
  await publishTunnelDNSRecord(config, route.host)
}

export async function hideCloudflareTunnelRoute(host: string): Promise<void> {
  const config = cloudflareConfig()
  const current = await tunnelConfiguration(config)
  await putTunnelConfiguration(config, {
    config: {
      ingress: removeIngressRule(current.config.ingress, host),
    },
  })
  await deleteDNSRecords(config, host)
}

export async function publishTailscaleDNSRecord(
  host: string,
  address: string
): Promise<void> {
  const config = cloudflareConfig()
  await upsertDNSRecord(config, host, {
    type: address.includes(":") ? "AAAA" : "A",
    name: host,
    content: address,
    ttl: 60,
    proxied: false,
    comment: "Ship Tailscale Gateway private route.",
  })
}

function cloudflareConfig(): CloudflareConfig {
  const apiToken = firstEnv("CLOUDFLARE_API_TOKEN", "CF_API_TOKEN")
  const accountID = firstEnv("CLOUDFLARE_ACCOUNT_ID", "CF_ACCOUNT_ID")
  const tunnelID = firstEnv("CLOUDFLARE_TUNNEL_ID")
  const zoneID = firstEnv("CLOUDFLARE_ZONE_ID", "CF_ZONE_ID")
  if (!apiToken || !accountID || !tunnelID || !zoneID) {
    throw new CloudflareTunnelSetupError(
      "cloudflare tunnel not ready; run: ship install"
    )
  }
  return {
    apiEndpoint:
      process.env.CLOUDFLARE_API_ENDPOINT ??
      "https://api.cloudflare.com/client/v4",
    apiToken,
    accountID,
    tunnelID,
    zoneID,
  }
}

function firstEnv(...names: readonly string[]): string {
  for (const name of names) {
    const value = process.env[name]
    if (value) {
      return value
    }
  }
  return ""
}

async function tunnelConfiguration(
  config: CloudflareConfig
): Promise<CloudflareTunnelConfiguration> {
  return cloudflareRequest(
    config,
    "GET",
    `/accounts/${encodeURIComponent(config.accountID)}/cfd_tunnel/${encodeURIComponent(config.tunnelID)}/configurations`,
    undefined,
    cloudflareTunnelConfiguration
  )
}

async function putTunnelConfiguration(
  config: CloudflareConfig,
  body: CloudflareTunnelConfiguration
): Promise<void> {
  await cloudflareRequest(
    config,
    "PUT",
    `/accounts/${encodeURIComponent(config.accountID)}/cfd_tunnel/${encodeURIComponent(config.tunnelID)}/configurations`,
    body,
    z.unknown()
  )
}

async function publishTunnelDNSRecord(
  config: CloudflareConfig,
  host: string
): Promise<void> {
  await upsertDNSRecord(config, host, {
    type: "CNAME",
    name: host,
    content: `${config.tunnelID}.cfargotunnel.com`,
    ttl: 1,
    proxied: true,
    comment: "Ship Cloudflare Tunnel public route.",
  })
}

async function upsertDNSRecord(
  config: CloudflareConfig,
  host: string,
  body: DNSRecordBody
): Promise<void> {
  const records = await listDNSRecords(config, host)
  let updated = false
  for (const record of records) {
    if (record.type !== body.type) {
      await deleteDNSRecord(config, record.id)
      continue
    }
    if (updated) {
      await deleteDNSRecord(config, record.id)
      continue
    }
    await cloudflareRequest(
      config,
      "PATCH",
      `/zones/${encodeURIComponent(config.zoneID)}/dns_records/${encodeURIComponent(record.id)}`,
      body,
      z.unknown()
    )
    updated = true
  }
  if (!updated) {
    await cloudflareRequest(
      config,
      "POST",
      `/zones/${encodeURIComponent(config.zoneID)}/dns_records`,
      body,
      z.unknown()
    )
  }
}

async function deleteDNSRecords(
  config: CloudflareConfig,
  host: string
): Promise<void> {
  const records = await listDNSRecords(config, host)
  await Promise.all(records.map((record) => deleteDNSRecord(config, record.id)))
}

async function listDNSRecords(
  config: CloudflareConfig,
  host: string
): Promise<readonly CloudflareRecord[]> {
  const response = await cloudflareRequest(
    config,
    "GET",
    `/zones/${encodeURIComponent(config.zoneID)}/dns_records?name=${encodeURIComponent(host)}&per_page=100`,
    undefined,
    z.array(cloudflareRecord)
  )
  return response
}

async function deleteDNSRecord(
  config: CloudflareConfig,
  recordID: string
): Promise<void> {
  await cloudflareRequest(
    config,
    "DELETE",
    `/zones/${encodeURIComponent(config.zoneID)}/dns_records/${encodeURIComponent(recordID)}`,
    undefined,
    z.unknown()
  )
}

async function cloudflareRequest<T>(
  config: CloudflareConfig,
  method: string,
  path: string,
  body: unknown,
  resultSchema: z.ZodType<T>
): Promise<T> {
  const response = await fetch(`${config.apiEndpoint}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${config.apiToken}`,
      "Content-Type": "application/json",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(30_000),
  })
  const payload: unknown = await response.json()
  const envelope = z
    .object({
      success: z.boolean(),
      errors: z.array(cloudflareEnvelopeError).default([]),
      result: resultSchema.optional(),
    })
    .parse(payload)
  if (!envelope.success) {
    const message =
      envelope.errors
        .map((item) => item.message)
        .filter((item) => item !== "")
        .join("; ") || `Cloudflare API ${response.status}`
    throw new Error(`cloudflare api failed: ${message}`)
  }
  if (envelope.result === undefined) {
    throw new Error("cloudflare api response had no result")
  }
  return envelope.result
}

function mergeIngressRules(
  existing: readonly CloudflareIngressRule[],
  route: CloudflareIngressRule
): CloudflareIngressRule[] {
  return [
    ...existing.filter(
      (item) => item.hostname !== route.hostname && !isCatchAll(item)
    ),
    route,
    { service: "http_status:404" },
  ]
}

function removeIngressRule(
  existing: readonly CloudflareIngressRule[],
  host: string
): CloudflareIngressRule[] {
  const rules = existing.filter(
    (item) => item.hostname !== host && !isCatchAll(item)
  )
  return [...rules, { service: "http_status:404" }]
}

function isCatchAll(rule: CloudflareIngressRule): boolean {
  return rule.hostname === undefined && rule.service.startsWith("http_status:")
}
