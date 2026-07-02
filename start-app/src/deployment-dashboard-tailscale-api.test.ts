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

describe("dashboard Tailscale exposure API", () => {
  it("removes the Tailscale Funnel ingress when switching back to Tailscale", async () => {
    const {
      execFileAsync,
      execFileWithInput,
      readMergedDeployments,
      resolveIPv4,
    } = await mockedDeploymentModules()
    process.env.CLOUDFLARE_API_TOKEN = "test-token"
    process.env.CLOUDFLARE_ACCOUNT_ID = "account-id"
    process.env.CLOUDFLARE_TUNNEL_ID = "tunnel-id"
    process.env.CLOUDFLARE_ZONE_ID = "zone-id"
    process.env.SHIP_DNS_PROPAGATION_STABLE_CHECKS = "1"
    process.env.SHIP_DNS_PROPAGATION_MINIMUM_WAIT_MS = "0"
    readMergedDeployments.mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: false,
      }),
    ])
    execFileAsync
      .mockResolvedValueOnce({ stdout: "100.124.154.47", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "200 100.124.154.47", stderr: "" })
      .mockResolvedValueOnce({ stdout: "200 100.124.154.47", stderr: "" })
    resolveIPv4.mockResolvedValue(["100.124.154.47"])
    let publishedTailscaleDNS = false
    const fetchMock = vi.fn(
      async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input)
        if (
          url.endsWith(
            "/accounts/account-id/cfd_tunnel/tunnel-id/configurations"
          )
        ) {
          if (init?.method === "PUT") {
            expect(String(init.body)).not.toContain("demo.example.com")
            return jsonResponse({ success: true, result: {} })
          }
          return jsonResponse({
            success: true,
            result: {
              config: {
                ingress: [
                  {
                    hostname: "demo.example.com",
                    service: "http://demo.ship-services.svc.cluster.local:80",
                  },
                  { service: "http_status:404" },
                ],
              },
            },
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
          expect(String(init?.body)).toContain('"type":"A"')
          expect(String(init?.body)).toContain('"name":"demo.example.com"')
          expect(String(init?.body)).toContain('"content":"100.124.154.47"')
          expect(String(init?.body)).toContain('"proxied":false')
          publishedTailscaleDNS = true
          return jsonResponse({ success: true, result: {} })
        }
        throw new Error(`unexpected request: ${url}`)
      }
    )
    vi.stubGlobal("fetch", fetchMock)

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "tailscale",
      })
    )
    const body = z
      .object({
        deployment: z.object({
          serviceName: z.string(),
          exposure: z.literal("tailscale"),
          tailscaleOnly: z.literal(true),
        }),
      })
      .parse(await response.json())

    expect(response.status).toBe(200)
    expect(body.deployment.serviceName).toBe("demo")
    expect(execFileWithInput).not.toHaveBeenCalled()
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
    expect(execFileAsync).toHaveBeenCalledWith("curl", [
      "-sS",
      "-L",
      "--fail",
      "--resolve",
      "demo.example.com:443:100.124.154.47",
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
    expect(fetchMock).toHaveBeenCalledWith(
      "https://api.cloudflare.com/client/v4/zones/zone-id/dns_records/record-id",
      expect.objectContaining({ method: "DELETE" })
    )
    expect(publishedTailscaleDNS).toBe(true)
  })

  it("keeps waiting when the Tailscale route still returns 404", async () => {
    const { execFileAsync, readMergedDeployments, resolveIPv4 } =
      await mockedDeploymentModules()
    process.env.CLOUDFLARE_API_TOKEN = "test-token"
    process.env.CLOUDFLARE_ACCOUNT_ID = "account-id"
    process.env.CLOUDFLARE_TUNNEL_ID = "tunnel-id"
    process.env.CLOUDFLARE_ZONE_ID = "zone-id"
    process.env.SHIP_DNS_PROPAGATION_STABLE_CHECKS = "1"
    process.env.SHIP_DNS_PROPAGATION_MINIMUM_WAIT_MS = "0"
    process.env.SHIP_ROUTE_ACCESS_TIMEOUT_MS = "0"
    process.env.SHIP_ROUTE_ACCESS_POLL_MS = "0"
    readMergedDeployments.mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: false,
      }),
    ])
    execFileAsync
      .mockResolvedValueOnce({ stdout: "100.124.154.47", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "404", stderr: "" })
    resolveIPv4.mockResolvedValue(["100.124.154.47"])
    vi.stubGlobal("fetch", cloudflareFetchForHide())

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "tailscale",
      })
    )
    const body = z.object({ error: z.string() }).parse(await response.json())

    expect(response.status).toBe(409)
    expect(body.error).toBe(
      "tailscale route is not reachable yet; check the Tailscale Gateway and DNS route"
    )
  })

  it("keeps waiting while public resolvers still return the internet route", async () => {
    const { execFileAsync, readMergedDeployments, resolveIPv4 } =
      await mockedDeploymentModules()
    process.env.CLOUDFLARE_API_TOKEN = "test-token"
    process.env.CLOUDFLARE_ACCOUNT_ID = "account-id"
    process.env.CLOUDFLARE_TUNNEL_ID = "tunnel-id"
    process.env.CLOUDFLARE_ZONE_ID = "zone-id"
    process.env.SHIP_DNS_PROPAGATION_TIMEOUT_MS = "0"
    process.env.SHIP_DNS_PROPAGATION_POLL_MS = "0"
    process.env.SHIP_DNS_PROPAGATION_STABLE_CHECKS = "1"
    process.env.SHIP_DNS_PROPAGATION_MINIMUM_WAIT_MS = "0"
    readMergedDeployments.mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: false,
      }),
    ])
    execFileAsync
      .mockResolvedValueOnce({ stdout: "100.124.154.47", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })
      .mockResolvedValueOnce({ stdout: "200", stderr: "" })
    resolveIPv4.mockResolvedValue(["104.21.55.34", "172.67.144.105"])
    vi.stubGlobal("fetch", cloudflareFetchForHide())

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "tailscale",
      })
    )
    const body = z.object({ error: z.string() }).parse(await response.json())

    expect(response.status).toBe(409)
    expect(body.error).toBe(
      "tailscale route DNS has not propagated yet; try again shortly"
    )
  })
})

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
  })
}

function cloudflareFetchForHide() {
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
        result: {
          config: {
            ingress: [
              {
                hostname: "demo.example.com",
                service: "http://demo.ship-services.svc.cluster.local:80",
              },
              { service: "http_status:404" },
            ],
          },
        },
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
