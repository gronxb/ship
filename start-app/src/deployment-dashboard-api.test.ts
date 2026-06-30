import { describe, expect, it, vi } from "vitest"
import { z } from "zod"

import { Route } from "./routes/api.deployments"

vi.mock("./lib/deployment-command", () => ({
  execFileAsync: vi.fn(),
}))

const RouteOptions = z.object({
  options: z.object({
    server: z.object({
      handlers: z.record(z.string(), z.unknown()),
    }),
  }),
})

describe("dashboard deployment API", () => {
  it("registers only the read-only deployment list handler", () => {
    const route = RouteOptions.parse(Route)

    expect(Object.keys(route.options.server.handlers).sort()).toEqual(["GET"])
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
