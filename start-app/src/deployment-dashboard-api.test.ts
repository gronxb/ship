import { afterEach, describe, expect, it, vi } from "vitest"
import { z } from "zod"

import {
  deploymentFixture,
  exposureRequest,
  mockedDeploymentModules,
} from "./deployment-dashboard-api-test-helpers"
import { changeDeploymentExposure } from "./lib/deployment-handlers"
import { listDeployments } from "./lib/deployment-list-handler"
import { Route } from "./routes/api.deployments"

const RouteOptions = z.object({
  options: z.object({
    server: z.object({
      handlers: z.record(z.string(), z.unknown()),
    }),
  }),
})

afterEach(() => {
  vi.clearAllMocks()
})

describe("dashboard deployment API", () => {
  it("registers deployment list and exposure update handlers", () => {
    const route = RouteOptions.parse(Route)

    expect(Object.keys(route.options.server.handlers).sort()).toEqual([
      "GET",
      "PATCH",
    ])
  })

  it("rejects malformed exposure updates before kubectl runs", async () => {
    const { execFileAsync, readMergedDeployments } =
      await mockedDeploymentModules()

    const response = await changeDeploymentExposure(
      exposureRequest({ serviceName: "../demo", exposure: "internet" })
    )

    expect(response.status).toBe(400)
    expect(readMergedDeployments).not.toHaveBeenCalled()
    expect(execFileAsync).not.toHaveBeenCalled()
  })

  it("rejects unknown deployment exposure updates before kubectl runs", async () => {
    const { execFileAsync, readMergedDeployments } =
      await mockedDeploymentModules()
    readMergedDeployments.mockResolvedValueOnce([])

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "internet",
      })
    )

    expect(response.status).toBe(404)
    expect(execFileAsync).not.toHaveBeenCalled()
  })

  it("hides the dashboard deployment when its host matches the forwarded request host", async () => {
    const { readMergedDeployments } = await mockedDeploymentModules()
    readMergedDeployments.mockResolvedValueOnce([
      {
        ...deploymentFixture({
          namespace: "ship-services",
          serviceName: "ship-dashboard",
          tailscaleOnly: true,
        }),
        host: "ship.example.com",
      },
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: true,
      }),
    ])

    const response = await listDeployments(
      new Request("http://localhost/api/deployments", {
        headers: { "x-forwarded-host": "ship.example.com" },
      })
    )
    const body = z
      .object({
        deployments: z.array(
          z.object({
            host: z.string(),
            serviceName: z.string(),
          })
        ),
      })
      .parse(await response.json())

    expect(body.deployments).toEqual([
      {
        host: "demo.example.com",
        serviceName: "demo",
      },
    ])
  })

  it("rejects internet exposure for the current dashboard host", async () => {
    const { execFileAsync, readMergedDeployments } =
      await mockedDeploymentModules()
    readMergedDeployments.mockResolvedValueOnce([
      {
        ...deploymentFixture({
          namespace: "ship-services",
          serviceName: "ops",
          tailscaleOnly: true,
        }),
        host: "ops.example.com",
      },
    ])

    const response = await changeDeploymentExposure(
      new Request("http://localhost/api/deployments", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-forwarded-host": "ops.example.com",
        },
        body: JSON.stringify({
          serviceName: "ops",
          namespace: "ship-services",
          exposure: "internet",
        }),
      })
    )
    const body = z.object({ error: z.string() }).parse(await response.json())

    expect(response.status).toBe(409)
    expect(body.error).toBe(
      "Ship dashboard cannot be exposed to the internet; keep it on Tailscale"
    )
    expect(execFileAsync).not.toHaveBeenCalled()
  })

  it("returns setup guidance when Cloudflare Tunnel is not ready", async () => {
    const { execFileWithInput, readMergedDeployments } =
      await mockedDeploymentModules()
    readMergedDeployments.mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: true,
      }),
    ])

    const response = await changeDeploymentExposure(
      exposureRequest({
        serviceName: "demo",
        namespace: "ship-services",
        exposure: "internet",
      })
    )
    const body = z.object({ error: z.string() }).parse(await response.json())

    expect(response.status).toBe(409)
    expect(body.error).toBe("cloudflare tunnel not ready; run: ship install")
    expect(execFileWithInput).not.toHaveBeenCalled()
  })
})
