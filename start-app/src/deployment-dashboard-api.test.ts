import { afterEach, describe, expect, it, vi } from "vitest"
import { z } from "zod"

import { changeDeploymentExposure } from "./lib/deployment-handlers"
import type * as DeploymentState from "./lib/deployment-state"
import { Route } from "./routes/api.deployments"

vi.mock("./lib/deployment-command", () => ({
  execFileAsync: vi.fn(),
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

  it("patches a tailscale route to internet exposure", async () => {
    const { execFileAsync } = await import("./lib/deployment-command")
    const { readMergedDeployments } = await import("./lib/deployment-state")
    const mockedExecFile = vi.mocked(execFileAsync)
    vi.mocked(readMergedDeployments).mockResolvedValueOnce([
      deploymentFixture({
        namespace: "ship-services",
        serviceName: "demo",
        tailscaleOnly: true,
      }),
    ])
    mockedExecFile.mockResolvedValueOnce({
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
    expect(mockedExecFile).toHaveBeenCalledWith("kubectl", [
      "patch",
      "httproute",
      "demo",
      "-n",
      "ship-services",
      "--type=merge",
      "-p",
      expect.stringContaining('"ship.local/exposure":"internet"'),
    ])
    expect(mockedExecFile).toHaveBeenCalledWith("kubectl", [
      "patch",
      "httproute",
      "demo",
      "-n",
      "ship-services",
      "--type=merge",
      "-p",
      expect.stringContaining('"name":"ship-internet"'),
    ])
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

  it("attempts container logs and returns a visible fallback when logs fail", async () => {
    const { execFileAsync } = await import("./lib/deployment-command")
    const { readClusterDeployments } = await import("./lib/deployment-cluster")
    const mockedExecFile = vi.mocked(execFileAsync)
    mockedExecFile
      .mockResolvedValueOnce({
        stdout: JSON.stringify({
          items: [
            {
              metadata: {
                name: "demo",
                namespace: "ship-services",
                labels: { "ship.local/exposure": "tailscale" },
              },
              spec: {
                hostnames: ["demo.example.com"],
                parentRefs: [{ name: "ship-tailscale" }],
              },
            },
          ],
        }),
        stderr: "",
      })
      .mockRejectedValueOnce(new Error("pods not ready"))

    const deployments = await readClusterDeployments()

    expect(mockedExecFile).toHaveBeenNthCalledWith(1, "kubectl", [
      "get",
      "httproute",
      "-n",
      "ship-services",
      "-o",
      "json",
    ])
    expect(mockedExecFile).toHaveBeenNthCalledWith(2, "kubectl", [
      "logs",
      "deployment/demo",
      "-n",
      "ship-services",
      "--tail=80",
      "--all-containers=true",
    ])
    expect(deployments[0]?.containerLogs).toBe(
      "Container logs unavailable: pods not ready"
    )
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
