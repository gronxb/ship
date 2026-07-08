import { z } from "zod"

import {
  CloudflareTunnelSetupError,
  exposeCloudflareTunnelRoute,
  hideCloudflareTunnelRoute,
  publishTailscaleDNSRecord,
} from "./cloudflare-tunnel"
import { execFileAsync } from "./deployment-command"
import { gatewayNamespace, tailscaleGatewayName } from "./deployment-config"
import { resolveIPv4 } from "./deployment-dns"
import { readMergedDeployments } from "./deployment-state"
import {
  dashboardHostFromRequest,
  isDashboardDeployment,
} from "./deployment-visibility"

const exposureUpdate = z.object({
  namespace: z
    .string()
    .regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/)
    .optional(),
  serviceName: z.string().regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/),
  exposure: z.enum(["tailscale", "internet"]),
})

const cloudflareTunnelSetupCommand = "ship install"
const dashboardInternetExposureError =
  "Ship dashboard cannot be exposed to the internet; keep it on Tailscale"
const dnsPropagationResolvers = ["1.1.1.1", "8.8.8.8", "9.9.9.9"] as const

class DNSPropagationError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "DNSPropagationError"
  }
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

  const { exposure, namespace: requestedNamespace, serviceName } = parsed.data
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
  if (
    exposure === "internet" &&
    isDashboardDeployment(deployment, dashboardHostFromRequest(request))
  ) {
    return Response.json(
      { error: dashboardInternetExposureError },
      { status: 409 }
    )
  }
  if (exposure === "tailscale") {
    if (deployment.tailscaleOnly) {
      return deploymentExposureResponse(
        serviceName,
        deployment.namespace,
        "tailscale"
      )
    }
    const tailscaleAddress = await switchDeploymentToTailscale(
      serviceName,
      deployment.namespace,
      deployment.host
    )
    return verifyAndReturnTailscaleDeployment(
      serviceName,
      deployment.namespace,
      deployment.host,
      tailscaleAddress
    )
  }

  if (!deployment.tailscaleOnly) {
    return deploymentExposureResponse(
      serviceName,
      deployment.namespace,
      "internet"
    )
  }

  try {
    await exposeCloudflareTunnelRoute({
      host: deployment.host,
      namespace: deployment.namespace,
      serviceName,
    })
  } catch (caught) {
    if (caught instanceof CloudflareTunnelSetupError) {
      return Response.json(
        {
          error: caught.message,
        },
        { status: 409 }
      )
    }
    throw caught
  }

  const tailscaleAddress = await tailscaleGatewayAddress()
  await deleteInternetIngress(serviceName, deployment.namespace)
  await labelHttpRouteAsInternet(serviceName, deployment.namespace)
  return verifyAndReturnInternetDeployment(
    serviceName,
    deployment.namespace,
    deployment.host,
    tailscaleAddress
  )
}

async function verifyAndReturnInternetDeployment(
  serviceName: string,
  namespace: string,
  host: string,
  tailscaleAddress: string
): Promise<Response> {
  try {
    await waitForDNSPropagation(host, {
      exposure: "internet",
      tailscaleAddress,
    })
    await verifyInternetAccess(host)
  } catch (caught) {
    if (!(caught instanceof Error)) {
      throw caught
    }
    await switchDeploymentToTailscale(
      serviceName,
      namespace,
      host,
      tailscaleAddress
    )
    return Response.json(
      {
        error:
          caught instanceof DNSPropagationError
            ? caught.message
            : `cloudflare tunnel hostname is not reachable from the internet; rerun: ${cloudflareTunnelSetupCommand}`,
      },
      { status: 409 }
    )
  }
  return Response.json({
    deployment: deploymentExposure(serviceName, namespace, "internet"),
  })
}

async function verifyAndReturnTailscaleDeployment(
  serviceName: string,
  namespace: string,
  host: string,
  tailscaleAddress: string
): Promise<Response> {
  try {
    await waitForDNSPropagation(host, {
      exposure: "tailscale",
      tailscaleAddress,
    })
    await verifyRouteAccess(host, {
      expectedRemoteAddress: tailscaleAddress,
    })
    await verifyRouteAccess(host, {
      expectedRemoteAddress: tailscaleAddress,
      resolveAddress: tailscaleAddress,
    })
  } catch (caught) {
    if (!(caught instanceof Error)) {
      throw caught
    }
    return Response.json(
      {
        error:
          caught instanceof DNSPropagationError
            ? caught.message
            : "tailscale route is not reachable yet; check the Tailscale Gateway and DNS route",
      },
      { status: 409 }
    )
  }
  return Response.json({
    deployment: deploymentExposure(serviceName, namespace, "tailscale"),
  })
}

function deploymentExposureResponse(
  serviceName: string,
  namespace: string,
  exposure: "tailscale" | "internet"
): Response {
  return Response.json({
    deployment: deploymentExposure(serviceName, namespace, exposure),
  })
}

function deploymentExposure(
  serviceName: string,
  namespace: string,
  exposure: "tailscale" | "internet"
) {
  return {
    serviceName,
    namespace,
    exposure,
    tailscaleOnly: exposure === "tailscale",
  }
}

async function switchDeploymentToTailscale(
  serviceName: string,
  namespace: string,
  host: string,
  gatewayAddress?: string
): Promise<string> {
  const tailscaleAddress = gatewayAddress ?? (await tailscaleGatewayAddress())
  await hideCloudflareTunnelRoute(host)
  await publishTailscaleDNSRecord(host, tailscaleAddress)
  await deleteInternetIngress(serviceName, namespace)
  await labelHttpRouteAsTailscale(serviceName, namespace)
  return tailscaleAddress
}

async function tailscaleGatewayAddress(): Promise<string> {
  const output = await execFileAsync("kubectl", [
    "get",
    "gateway",
    tailscaleGatewayName,
    "-n",
    gatewayNamespace,
    "-o",
    "jsonpath={.status.addresses[0].value}",
  ])
  const address = output.stdout.trim()
  if (address === "") {
    throw new Error(
      `gateway ${gatewayNamespace}/${tailscaleGatewayName} has no published address`
    )
  }
  return address
}

async function deleteInternetIngress(
  serviceName: string,
  namespace: string
): Promise<void> {
  await execFileAsync("kubectl", [
    "delete",
    "ingress",
    serviceName,
    "-n",
    namespace,
    "--ignore-not-found",
  ])
}

async function labelHttpRouteAsTailscale(
  serviceName: string,
  namespace: string
): Promise<void> {
  await execFileAsync("kubectl", [
    "label",
    "httproute",
    serviceName,
    "-n",
    namespace,
    "ship.local/exposure=tailscale",
    "ship.local/tailscale-only=true",
    "--overwrite",
  ])
}

async function labelHttpRouteAsInternet(
  serviceName: string,
  namespace: string
): Promise<void> {
  await execFileAsync("kubectl", [
    "label",
    "httproute",
    serviceName,
    "-n",
    namespace,
    "ship.local/exposure=internet",
    "ship.local/tailscale-only=false",
    "--overwrite",
  ])
}

async function verifyInternetAccess(host: string): Promise<void> {
  try {
    await verifyRouteAccess(host)
  } catch (caught) {
    if (!(caught instanceof Error)) {
      throw caught
    }
    throw new Error(
      "public Cloudflare hostname did not return an HTTP response"
    )
  }
}

async function waitForDNSPropagation(
  host: string,
  expected: {
    readonly exposure: "tailscale" | "internet"
    readonly tailscaleAddress: string
  }
): Promise<void> {
  const started = Date.now()
  const deadline = Date.now() + dnsPropagationTimeoutMs()
  let consecutiveMatches = 0
  for (;;) {
    const answers = await Promise.all(
      dnsPropagationResolvers.map(async (server) => ({
        server,
        addresses: await resolveIPv4(host, server).catch(() => []),
      }))
    )
    if (dnsAnswersMatch(answers, expected)) {
      consecutiveMatches += 1
      if (
        consecutiveMatches >= dnsPropagationStableChecks() &&
        Date.now() - started >= dnsPropagationMinimumWaitMs()
      ) {
        return
      }
    } else {
      consecutiveMatches = 0
    }
    if (Date.now() >= deadline) {
      throw new DNSPropagationError(
        expected.exposure === "tailscale"
          ? "tailscale route DNS has not propagated yet; try again shortly"
          : "internet route DNS has not propagated yet; try again shortly"
      )
    }
    await sleep(
      Math.min(dnsPropagationPollMs(), Math.max(0, deadline - Date.now()))
    )
  }
}

function dnsAnswersMatch(
  answers: readonly {
    readonly server: string
    readonly addresses: readonly string[]
  }[],
  expected: {
    readonly exposure: "tailscale" | "internet"
    readonly tailscaleAddress: string
  }
): boolean {
  if (expected.exposure === "tailscale") {
    return answers.every((answer) =>
      answer.addresses.includes(expected.tailscaleAddress)
    )
  }
  return answers.every(
    (answer) =>
      answer.addresses.length > 0 &&
      !answer.addresses.includes(expected.tailscaleAddress)
  )
}

async function sleep(milliseconds: number): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, milliseconds))
}

function dnsPropagationTimeoutMs(): number {
  return positiveNumber(process.env.SHIP_DNS_PROPAGATION_TIMEOUT_MS, 480_000)
}

function dnsPropagationPollMs(): number {
  return positiveNumber(process.env.SHIP_DNS_PROPAGATION_POLL_MS, 5_000)
}

function dnsPropagationStableChecks(): number {
  return positiveNumber(process.env.SHIP_DNS_PROPAGATION_STABLE_CHECKS, 2)
}

function dnsPropagationMinimumWaitMs(): number {
  return positiveNumber(
    process.env.SHIP_DNS_PROPAGATION_MINIMUM_WAIT_MS,
    300_000
  )
}

function positiveNumber(value: string | undefined, fallback: number): number {
  if (value === undefined) {
    return fallback
  }
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : fallback
}

async function verifyRouteAccess(
  host: string,
  options: {
    readonly expectedRemoteAddress?: string
    readonly resolveAddress?: string
  } = {}
): Promise<void> {
  const deadline = Date.now() + routeAccessTimeoutMs()
  for (;;) {
    if (await probeRouteAccess(host, options)) {
      return
    }
    if (Date.now() >= deadline) {
      throw new Error("route did not return an HTTP response")
    }
    await sleep(
      Math.min(routeAccessPollMs(), Math.max(0, deadline - Date.now()))
    )
  }
}

async function probeRouteAccess(
  host: string,
  options: {
    readonly expectedRemoteAddress?: string
    readonly resolveAddress?: string
  }
): Promise<boolean> {
  try {
    const output = await execFileAsync("curl", [
      "-sS",
      "-L",
      "--fail",
      ...(options.resolveAddress === undefined
        ? []
        : ["--resolve", `${host}:443:${options.resolveAddress}`]),
      "--connect-timeout",
      "10",
      "--max-time",
      "15",
      "-o",
      "/dev/null",
      "-w",
      "%{http_code} %{remote_ip}",
      `https://${host}`,
    ])
    const [statusText, remoteAddress] = output.stdout.trim().split(/\s+/, 2)
    const status = Number.parseInt(statusText, 10)
    return (
      Number.isInteger(status) &&
      status >= 200 &&
      status <= 399 &&
      (options.expectedRemoteAddress === undefined ||
        remoteAddress === options.expectedRemoteAddress)
    )
  } catch {
    return false
  }
}

function routeAccessTimeoutMs(): number {
  return positiveNumber(process.env.SHIP_ROUTE_ACCESS_TIMEOUT_MS, 420_000)
}

function routeAccessPollMs(): number {
  return positiveNumber(process.env.SHIP_ROUTE_ACCESS_POLL_MS, 5_000)
}
