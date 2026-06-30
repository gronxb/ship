// @vitest-environment jsdom

import {
  cleanup,
  render,
  screen,
} from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { DeploymentDashboard } from "./components/dashboard/deployment-dashboard"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("dashboard surface", () => {
  it("renders deployed containers as read-only cards with logs", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        return new Response(
          JSON.stringify({
            deployments: [
              {
                serviceName: "demo",
                host: "demo.example.com",
                image: "ship/demo:latest",
                namespace: "ship-services",
                port: 3000,
                exposure: "tailscale",
                tailscaleOnly: true,
                dryRun: false,
                commands: ["kubectl apply -f -"],
                manifest: "kind: Deployment",
                containerLogs: "listening on 3000",
                createdAt: "2026-06-30T00:00:00Z",
              },
            ],
          }),
          { headers: { "Content-Type": "application/json" } }
        )
      })
    )

    await renderDashboardAt("/")

    expect(screen.getByText("demo")).toBeDefined()
    expect(screen.getByText("demo.example.com")).toBeDefined()
    expect(screen.getByRole("button", { name: "Refresh" })).toBeDefined()
    expect(screen.queryByRole("button", { name: /deploy/i })).toBeNull()
    expect(screen.queryByText(/[ㄱ-ㅎㅏ-ㅣ가-힣]/)).toBeNull()
    expect(renderedTextIncludes("GET https://demo.example.com")).toBe(true)

    cleanup()
    await renderDashboardAt("/?tab=terminal")
    expect(screen.getByRole("tab", { selected: true }).textContent).toContain(
      "Terminal"
    )
    expect(renderedTextIncludes("kubectl apply -f -")).toBe(true)
    expect(renderedTextIncludes("listening on 3000")).toBe(true)

    cleanup()
    await renderDashboardAt("/?tab=details")
    expect(renderedTextIncludes("kind: Deployment")).toBe(true)
  })
})

async function renderDashboardAt(path: string): Promise<void> {
  window.history.replaceState(null, "", path)
  render(<DeploymentDashboard />)
  await screen.findByText("demo")
}

function renderedTextIncludes(text: string): boolean {
  return document.body.textContent.includes(text)
}
