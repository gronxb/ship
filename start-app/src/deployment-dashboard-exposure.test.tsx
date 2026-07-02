// @vitest-environment jsdom

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
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

    expect(screen.getByRole("status").textContent).toContain(
      "Waiting for DNS and route propagation for demo.example.com..."
    )
    expect(screen.getByRole("status").textContent).toContain(
      "Ship is waiting for public DNS caches to leave the Tailscale address"
    )
    expect(screen.getByRole("status").textContent).toContain(
      "This usually takes about 5 minutes"
    )

    await act(async () => {
      resolvePatch(
        new Response(
          JSON.stringify({
            deployment: {
              serviceName: "demo",
              namespace: "ship-services",
              exposure: "internet",
              tailscaleOnly: false,
            },
          }),
          { headers: { "Content-Type": "application/json" } }
        )
      )
      await Promise.resolve()
    })
  })

  it("keeps the deployment card visible while the exposed route propagates", async () => {
    const initialDeployment = deployment()
    let resolvePatch = (_response: Response): void => {}
    const fetchMock = vi.fn(
      async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "PATCH") {
          return new Promise<Response>((resolve) => {
            resolvePatch = resolve
          })
        }
        return new Response(JSON.stringify({ deployments: [] }), {
          headers: { "Content-Type": "application/json" },
        })
      }
    )
    vi.stubGlobal("fetch", fetchMock)

    window.history.replaceState(null, "", "/")
    render(<DeploymentDashboard initialDeployments={[initialDeployment]} />)
    fireEvent.click(screen.getByRole("button", { name: "Expose to internet" }))

    expect(
      screen.getByText(
        "Waiting for DNS and route propagation for demo.example.com..."
      )
    ).toBeDefined()
    expect(
      screen.getByText(
        "This usually takes about 5 minutes because Cloudflare proxied records use a 300 second cache window."
      )
    ).toBeDefined()
    expect(
      screen.getByRole("button", { name: "Waiting" }).getAttribute("disabled")
    ).toBe("")
    expect(screen.getByRole("button", { name: "Refresh" })).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      resolvePatch(
        new Response(
          JSON.stringify({
            deployment: {
              serviceName: "demo",
              namespace: "ship-services",
              exposure: "internet",
              tailscaleOnly: false,
            },
          }),
          { headers: { "Content-Type": "application/json" } }
        )
      )
      await Promise.resolve()
    })

    expect(screen.getByText("Internet network")).toBeDefined()
    expect(
      screen.getByRole("button", { name: "Use Tailscale only" })
    ).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it("keeps Tailscale-only route propagation visible until PATCH probe finishes", async () => {
    const initialDeployment = deployment({
      exposure: "internet",
      host: "demo.example.com",
      tailscaleOnly: false,
    })
    let resolvePatch = (_response: Response): void => {}
    const fetchMock = vi.fn(
      async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "PATCH") {
          return new Promise<Response>((resolve) => {
            resolvePatch = resolve
          })
        }
        return new Response(JSON.stringify({ deployments: [] }), {
          headers: { "Content-Type": "application/json" },
        })
      }
    )
    vi.stubGlobal("fetch", fetchMock)

    window.history.replaceState(null, "", "/")
    render(<DeploymentDashboard initialDeployments={[initialDeployment]} />)
    fireEvent.click(screen.getByRole("button", { name: "Use Tailscale only" }))

    expect(
      screen.getByText(
        "Waiting for Tailscale route propagation for demo.example.com..."
      )
    ).toBeDefined()
    expect(
      screen.getByText(
        "Ship is waiting for DNS caches to leave Cloudflare, then probing the Tailscale Gateway before it reports success."
      )
    ).toBeDefined()
    expect(
      screen.getByText(
        "This usually takes 5 to 7 minutes while recursive DNS and the local route cache converge."
      )
    ).toBeDefined()
    expect(screen.getByRole("button", { name: "Waiting" })).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      resolvePatch(
        new Response(
          JSON.stringify({
            deployment: {
              serviceName: "demo",
              namespace: "ship-services",
              exposure: "tailscale",
              tailscaleOnly: true,
            },
          }),
          { headers: { "Content-Type": "application/json" } }
        )
      )
      await Promise.resolve()
    })

    expect(screen.getAllByText("Tailscale-only").length).toBeGreaterThan(0)
    expect(
      screen.getByRole("button", { name: "Expose to internet" })
    ).toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it("shows the ship install setup command when Cloudflare exposure is not ready", async () => {
    const initialDeployment = deployment()
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "PATCH") {
          return new Response(
            JSON.stringify({
              error: "cloudflare tunnel not ready; run: ship install",
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
      await screen.findByText("cloudflare tunnel not ready; run: ship install")
    ).toBeDefined()
    expect(document.body.textContent.includes('{"error"')).toBe(false)
  })

  it("switches an internet deployment back to Tailscale-only", async () => {
    const initialDeployment = deployment({
      exposure: "internet",
      host: "demo.example.com",
      tailscaleOnly: false,
    })
    const fetchMock = vi.fn(
      async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "PATCH") {
          return new Response(
            JSON.stringify({
              deployment: {
                serviceName: "demo",
                exposure: "tailscale",
                tailscaleOnly: true,
              },
            }),
            { headers: { "Content-Type": "application/json" } }
          )
        }
        return new Response(
          JSON.stringify({ deployments: [initialDeployment] }),
          {
            headers: { "Content-Type": "application/json" },
          }
        )
      }
    )
    vi.stubGlobal("fetch", fetchMock)

    window.history.replaceState(null, "", "/")
    render(<DeploymentDashboard initialDeployments={[initialDeployment]} />)
    fireEvent.click(screen.getByRole("button", { name: "Use Tailscale only" }))

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/deployments",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({
          serviceName: "demo",
          namespace: "ship-services",
          exposure: "tailscale",
        }),
      })
    )
  })

  it("labels internet deployments as internet network routes", () => {
    const initialDeployment = deployment({
      exposure: "internet",
      host: "demo.example.com",
      tailscaleOnly: false,
    })

    window.history.replaceState(null, "", "/")
    render(<DeploymentDashboard initialDeployments={[initialDeployment]} />)

    expect(screen.getByText("Internet network")).toBeDefined()
  })
})

function deployment(overrides: Partial<Deployment> = {}): Deployment {
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
    ...overrides,
  }
}
