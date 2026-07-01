import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("./lib/deployment-command", () => ({
  execFileAsync: vi.fn(),
}))

afterEach(() => {
  vi.clearAllMocks()
})

describe("cluster deployments", () => {
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
      .mockResolvedValueOnce({
        stdout: JSON.stringify({ items: [] }),
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
      "get",
      "ingress",
      "-n",
      "ship-services",
      "-l",
      "ship.local/exposure=internet",
      "-o",
      "json",
    ])
    expect(mockedExecFile).toHaveBeenNthCalledWith(3, "kubectl", [
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

  it("shows internet exposure from a Tailscale Funnel ingress", async () => {
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
      .mockResolvedValueOnce({
        stdout: JSON.stringify({
          items: [
            {
              metadata: {
                name: "demo",
                namespace: "ship-services",
                labels: {
                  "app.kubernetes.io/name": "demo",
                  "ship.local/exposure": "internet",
                },
              },
              status: {
                loadBalancer: {
                  ingress: [{ hostname: "demo.tailnet.ts.net" }],
                },
              },
            },
          ],
        }),
        stderr: "",
      })
      .mockResolvedValueOnce({ stdout: "", stderr: "" })

    const deployments = await readClusterDeployments()

    expect(deployments[0]?.exposure).toBe("internet")
    expect(deployments[0]?.tailscaleOnly).toBe(false)
    expect(deployments[0]?.host).toBe("demo.tailnet.ts.net")
  })
})
