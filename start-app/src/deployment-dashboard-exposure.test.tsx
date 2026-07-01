// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { DeploymentDashboard } from "./components/dashboard/deployment-dashboard"
import type { Deployment } from "./components/dashboard/types"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("dashboard exposure errors", () => {
  it("shows progress while internet exposure is pending", async () => {
    const initialDeployment = deployment()
    let resolvePatch = (_response: Response): void => {}
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "PATCH") {
          return new Promise<Response>((resolve) => {
            resolvePatch = resolve
          })
        }
        return new Response(
          JSON.stringify({ deployments: [initialDeployment] }),
          {
            headers: { "Content-Type": "application/json" },
          }
        )
      })
    )

    window.history.replaceState(null, "", "/")
    render(<DeploymentDashboard initialDeployments={[initialDeployment]} />)
    fireEvent.click(screen.getByRole("button", { name: "Expose to internet" }))

    expect(screen.getByRole("status").textContent).toBe(
      "Exposing demo to the internet..."
    )

    resolvePatch(
      new Response(
        JSON.stringify({
          deployment: {
            serviceName: "demo",
            exposure: "internet",
            tailscaleOnly: false,
          },
        }),
        { headers: { "Content-Type": "application/json" } }
      )
    )
  })

  it("shows the Tailscale Funnel setup command when exposure is not ready", async () => {
    const initialDeployment = deployment()
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "PATCH") {
          return new Response(
            JSON.stringify({
              error:
                "tailscale funnel not ready; run: cd deploy-system && ./deploy-internet-gateway.sh",
            }),
            {
              status: 409,
              headers: { "Content-Type": "application/json" },
            }
          )
        }
        return new Response(
          JSON.stringify({ deployments: [initialDeployment] }),
          {
            headers: { "Content-Type": "application/json" },
          }
        )
      })
    )

    window.history.replaceState(null, "", "/")
    render(<DeploymentDashboard initialDeployments={[initialDeployment]} />)
    fireEvent.click(screen.getByRole("button", { name: "Expose to internet" }))

    expect(
      await screen.findByText(
        "tailscale funnel not ready; run: cd deploy-system && ./deploy-internet-gateway.sh"
      )
    ).toBeDefined()
    expect(document.body.textContent.includes('{"error"')).toBe(false)
  })
})

function deployment(): Deployment {
  return {
    serviceName: "demo",
    host: "demo.example.com",
    image: "ship/demo:latest",
    namespace: "ship-services",
    port: 3000,
    exposure: "tailscale",
    tailscaleOnly: true,
    dryRun: false,
    commands: [],
    manifest: "",
    containerLogs: "",
  }
}
