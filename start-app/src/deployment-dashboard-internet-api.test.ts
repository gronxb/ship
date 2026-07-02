import { afterEach, describe, expect, it, vi } from "vitest"
import { z } from "zod"

import {
  deploymentFixture,
  exposureRequest,
  mockedDeploymentModules,
} from "./deployment-dashboard-api-test-helpers"
import { changeDeploymentExposure } from "./lib/deployment-handlers"

afterEach(() => {
  vi.clearAllMocks()
  vi.unstubAllGlobals()
  delete process.env.CLOUDFLARE_API_TOKEN
  delete process.env.CLOUDFLARE_ACCOUNT_ID
  delete process.env.CLOUDFLARE_TUNNEL_ID
  delete process.env.CLOUDFLARE_ZONE_ID
  delete process.env.SHIP_DNS_PROPAGATION_TIMEOUT_MS
  delete process.env.SHIP_DNS_PROPAGATION_POLL_MS
  delete process.env.SHIP_DNS_PROPAGATION_STABLE_CHECKS
  delete process.env.SHIP_DNS_PROPAGATION_MINIMUM_WAIT_MS
  delete process.env.SHIP_ROUTE_ACCESS_TIMEOUT_MS
  delete process.env.SHIP_ROUTE_ACCESS_POLL_MS
})

describe("dashboard internet exposure API", () => {
  it("publishes the same hostname through Cloudflare Tunnel for internet exposure", async () => {
    const {
      execFileAsync,
      execFileWithInput,
      readMergedDeployments,
      resolveIPv4,
    } = await mockedDeploymentModules()
    stubCloudflareEnv()
    process.env.SHIP_DNS_PROPAGATION_STABLE_CHECKS = "1"
    process.env.SHIP_DNS_PROPAGATION_MINIMUM_WAIT_MS = "0"
    readMergedDeployments.mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: true,
      }),
    ])
    execFileAsync
      .mockResolvedValueOnce({ stdout: "100.124.154.47", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "200", stderr: "" })
    resolveIPv4.mockResolvedValue(["104.21.55.34", "172.67.144.105"])
    const requests: string[] = []
    const fetchMock = vi.fn(
      async (input: RequestInfo | URL, init?: RequestInit) => {
        requests.push(`${init?.method ?? "GET"} ${String(input)}`)
        if (
          String(input).endsWith(
            "/accounts/account-id/cfd_tunnel/tunnel-id/configurations"
          ) &&
          init?.method === "GET"
        ) {
          return jsonResponse({
            success: true,
            result: { config: { ingress: [{ service: "http_status:404" }] } },
          })
        }
        if (
          String(input).endsWith(
            "/accounts/account-id/cfd_tunnel/tunnel-id/configurations"
          ) &&
          init?.method === "PUT"
        ) {
          expect(String(init.body)).toContain('"hostname":"demo.example.com"')
          expect(String(init.body)).toContain(
            '"service":"http://demo.ship-services.svc.cluster.local:80"'
          )
          return jsonResponse({ success: true, result: {} })
        }
        if (
          String(input).endsWith(
            "/zones/zone-id/dns_records?name=demo.example.com&per_page=100"
          )
        ) {
          return jsonResponse({
            success: true,
            result: [{ id: "old-a", type: "A" }],
          })
        }
        if (String(input).endsWith("/zones/zone-id/dns_records/old-a")) {
          return jsonResponse({ success: true, result: {} })
        }
        if (String(input).endsWith("/zones/zone-id/dns_records")) {
          expect(String(init?.body)).toContain('"type":"CNAME"')
          expect(String(init?.body)).toContain('"name":"demo.example.com"')
          expect(String(init?.body)).toContain(
            '"content":"tunnel-id.cfargotunnel.com"'
          )
          expect(String(init?.body)).toContain('"proxied":true')
          return jsonResponse({ success: true, result: {} })
        }
        throw new Error(`unexpected request: ${String(input)}`)
      }
    )
    vi.stubGlobal("fetch", fetchMock)

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "internet",
      })
    )
    const body = z
      .object({
        deployment: z.object({
          serviceName: z.string(),
          exposure: z.literal("internet"),
          tailscaleOnly: z.literal(false),
        }),
      })
      .parse(await response.json())

    expect(response.status).toBe(200)
    expect(body.deployment.serviceName).toBe("demo")
    expect(execFileAsync).toHaveBeenCalledWith("kubectl", [
      "delete",
      "ingress",
      "demo",
      "-n",
      "ship-services",
      "--ignore-not-found",
    ])
    expect(execFileAsync).toHaveBeenCalledWith("kubectl", [
      "label",
      "httproute",
      "demo",
      "-n",
      "ship-services",
      "ship.local/exposure=internet",
      "ship.local/tailscale-only=false",
      "--overwrite",
    ])
    expect(execFileAsync).toHaveBeenCalledWith("curl", [
      "-sS",
      "-L",
      "--fail",
      "--connect-timeout",
      "10",
      "--max-time",
      "15",
      "-o",
      "/dev/null",
      "-w",
      "%{http_code} %{remote_ip}",
      "https://demo.example.com",
    ])
    expect(execFileWithInput).not.toHaveBeenCalled()
    expect(requests).toContain(
      "POST https://api.cloudflare.com/client/v4/zones/zone-id/dns_records"
    )
  })

  it("does not require public reachability for an existing internet deployment", async () => {
    const { execFileAsync, readMergedDeployments } =
      await mockedDeploymentModules()
    readMergedDeployments.mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: false,
      }),
    ])

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "internet",
      })
    )
    const body = z
      .object({
        deployment: z.object({
          serviceName: z.string(),
          exposure: z.literal("internet"),
          tailscaleOnly: z.literal(false),
        }),
      })
      .parse(await response.json())

    expect(response.status).toBe(200)
    expect(body.deployment.serviceName).toBe("demo")
    expect(execFileAsync).not.toHaveBeenCalled()
  })

  it("rolls back internet exposure when the public Cloudflare hostname is unreachable", async () => {
    const {
      execFileAsync,
      execFileWithInput,
      readMergedDeployments,
      resolveIPv4,
    } = await mockedDeploymentModules()
    stubCloudflareEnv()
    process.env.SHIP_DNS_PROPAGATION_STABLE_CHECKS = "1"
    process.env.SHIP_DNS_PROPAGATION_MINIMUM_WAIT_MS = "0"
    process.env.SHIP_ROUTE_ACCESS_TIMEOUT_MS = "0"
    process.env.SHIP_ROUTE_ACCESS_POLL_MS = "0"
    readMergedDeployments.mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: true,
      }),
    ])
    execFileAsync
      .mockResolvedValueOnce({ stdout: "100.124.154.47", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockRejectedValueOnce(new Error("Could not resolve host"))
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
    resolveIPv4.mockResolvedValue(["104.21.55.34", "172.67.144.105"])
    vi.stubGlobal("fetch", cloudflareFetchForExposeAndHide())

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "internet",
      })
    )
    const body = z.object({ error: z.string() }).parse(await response.json())

    expect(response.status).toBe(409)
    expect(body.error).toBe(
      "cloudflare tunnel hostname is not reachable from the internet; rerun: ship install"
    )
    expectTailscaleRollback(execFileAsync)
    expect(execFileWithInput).not.toHaveBeenCalled()
  })

  it("returns an existing internet deployment without rechecking propagation", async () => {
    const { execFileAsync, execFileWithInput, readMergedDeployments } =
      await mockedDeploymentModules()
    readMergedDeployments.mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: false,
      }),
    ])

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "internet",
      })
    )
    const body = z
      .object({
        deployment: z.object({
          serviceName: z.string(),
          exposure: z.literal("internet"),
          tailscaleOnly: z.literal(false),
        }),
      })
      .parse(await response.json())

    expect(response.status).toBe(200)
    expect(body.deployment.serviceName).toBe("demo")
    expect(execFileAsync).not.toHaveBeenCalled()
    expect(execFileWithInput).not.toHaveBeenCalled()
  })
})

function stubCloudflareEnv(): void {
  process.env.CLOUDFLARE_API_TOKEN = "test-token"
  process.env.CLOUDFLARE_ACCOUNT_ID = "account-id"
  process.env.CLOUDFLARE_TUNNEL_ID = "tunnel-id"
  process.env.CLOUDFLARE_ZONE_ID = "zone-id"
}

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
  })
}

function cloudflareFetchForExposeAndHide(): ReturnType<typeof vi.fn> {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input)
    if (
      url.endsWith("/accounts/account-id/cfd_tunnel/tunnel-id/configurations")
    ) {
      if (init?.method === "PUT") {
        return jsonResponse({ success: true, result: {} })
      }
      return jsonResponse({
        success: true,
        result: { config: { ingress: [{ service: "http_status:404" }] } },
      })
    }
    if (
      url.endsWith(
        "/zones/zone-id/dns_records?name=demo.example.com&per_page=100"
      )
    ) {
      return jsonResponse({
        success: true,
        result: [{ id: "record-id", type: "CNAME" }],
      })
    }
    if (url.endsWith("/zones/zone-id/dns_records/record-id")) {
      return jsonResponse({ success: true, result: {} })
    }
    if (url.endsWith("/zones/zone-id/dns_records")) {
      return jsonResponse({ success: true, result: {} })
    }
    throw new Error(`unexpected request: ${url}`)
  })
}

function expectTailscaleRollback(
  execFileAsync: ReturnType<typeof vi.fn>
): void {
  expect(execFileAsync).toHaveBeenCalledWith("kubectl", [
    "delete",
    "ingress",
    "demo",
    "-n",
    "ship-services",
    "--ignore-not-found",
  ])
  expect(execFileAsync).toHaveBeenCalledWith("kubectl", [
    "label",
    "httproute",
    "demo",
    "-n",
    "ship-services",
    "ship.local/exposure=tailscale",
    "ship.local/tailscale-only=true",
    "--overwrite",
  ])
}
