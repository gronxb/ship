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
  it("shows the internet Gateway setup command when exposure is not ready", async () => {
    const initialDeployment = deployment()
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "PATCH") {
          return new Response(
            JSON.stringify({
              error:
                "internet gateway not found; run: cd deploy-system && ./deploy-internet-gateway.sh",
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
        "internet gateway not found; run: cd deploy-system && ./deploy-internet-gateway.sh"
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
