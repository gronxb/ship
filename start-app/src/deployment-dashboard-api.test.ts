import { afterEach, describe, expect, it, vi } from "vitest"
import { z } from "zod"

import { changeDeploymentExposure } from "./lib/deployment-handlers"
import type * as DeploymentState from "./lib/deployment-state"
import { Route } from "./routes/api.deployments"

vi.mock("./lib/deployment-command", () => ({
  execFileAsync: vi.fn(),
  execFileWithInput: vi.fn(),
}))
vi.mock("./lib/deployment-state", async (importOriginal) => {
  const actual = await importOriginal<typeof DeploymentState>()
  return {
    ...actual,
    readMergedDeployments: vi.fn(),
  }
})

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

  it("applies a Tailscale Funnel ingress for internet exposure", async () => {
    const { execFileAsync, execFileWithInput } =
      await import("./lib/deployment-command")
    const { readMergedDeployments } = await import("./lib/deployment-state")
    const mockedExecFile = vi.mocked(execFileAsync)
    const mockedExecFileWithInput = vi.mocked(execFileWithInput)
    vi.mocked(readMergedDeployments).mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: true,
      }),
    ])
    mockedExecFileWithInput.mockResolvedValueOnce({
      stdout: "",
      stderr: "",
    })

    const response = await changeDeploymentExposure(
      new Request("http://ship.local/api/deployments", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          serviceName: "demo",
          namespace: "ship-services",
          exposure: "internet",
        }),
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
    expect(mockedExecFile).not.toHaveBeenCalled()
    expect(mockedExecFileWithInput).toHaveBeenCalledWith(
      "kubectl",
      ["apply", "-f", "-"],
      expect.stringContaining("kind: Ingress")
    )
    expect(mockedExecFileWithInput).toHaveBeenCalledWith(
      "kubectl",
      ["apply", "-f", "-"],
      expect.stringContaining('tailscale.com/funnel: "true"')
    )
    expect(mockedExecFileWithInput).toHaveBeenCalledWith(
      "kubectl",
      ["apply", "-f", "-"],
      expect.stringContaining("ingressClassName: tailscale")
    )
  })

  it("returns setup guidance when Tailscale Funnel is not ready", async () => {
    const { execFileWithInput } = await import("./lib/deployment-command")
    const { readMergedDeployments } = await import("./lib/deployment-state")
    const mockedExecFileWithInput = vi.mocked(execFileWithInput)
    vi.mocked(readMergedDeployments).mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: true,
      }),
    ])
    mockedExecFileWithInput.mockRejectedValueOnce(new Error("funnel disabled"))

    const response = await changeDeploymentExposure(
      new Request("http://ship.local/api/deployments", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          serviceName: "demo",
          namespace: "ship-services",
          exposure: "internet",
        }),
      })
    )
    const body = z.object({ error: z.string() }).parse(await response.json())

    expect(response.status).toBe(409)
    expect(body.error).toBe(
      "tailscale funnel not ready; run: cd deploy-system && ./deploy-internet-gateway.sh"
    )
    expect(mockedExecFileWithInput).toHaveBeenCalledTimes(1)
  })

  it("rejects malformed exposure updates before kubectl runs", async () => {
    const { execFileAsync } = await import("./lib/deployment-command")
    const { readMergedDeployments } = await import("./lib/deployment-state")
    const mockedExecFile = vi.mocked(execFileAsync)

    const response = await changeDeploymentExposure(
      new Request("http://ship.local/api/deployments", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ serviceName: "../demo", exposure: "internet" }),
      })
    )

    expect(response.status).toBe(400)
    expect(readMergedDeployments).not.toHaveBeenCalled()
    expect(mockedExecFile).not.toHaveBeenCalled()
  })

  it("rejects unknown deployment exposure updates before kubectl runs", async () => {
    const { execFileAsync } = await import("./lib/deployment-command")
    const { readMergedDeployments } = await import("./lib/deployment-state")
    const mockedExecFile = vi.mocked(execFileAsync)
    vi.mocked(readMergedDeployments).mockResolvedValueOnce([])

    const response = await changeDeploymentExposure(
      new Request("http://ship.local/api/deployments", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          serviceName: "demo",
          namespace: "ship-services",
          exposure: "internet",
        }),
      })
    )

    expect(response.status).toBe(404)
    expect(mockedExecFile).not.toHaveBeenCalled()
  })
})

function deploymentFixture({
  namespace,
  serviceName,
  tailscaleOnly,
}: {
  readonly namespace: string
  readonly serviceName: string
  readonly tailscaleOnly: boolean
}) {
  return {
    serviceName,
    host: `${serviceName}.example.com`,
    image: "cluster-managed",
    namespace,
    dockerfilePath: "",
    contextDir: "",
    port: 0,
    exposure: tailscaleOnly ? ("tailscale" as const) : ("internet" as const),
    tailscaleOnly,
    dryRun: false,
    commands: [],
    manifest: "",
    containerLogs: "",
  }
}
