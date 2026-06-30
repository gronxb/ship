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
  it("renders initial deployments before the browser refresh completes", () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        return new Response(JSON.stringify({ deployments: [] }), {
          headers: { "Content-Type": "application/json" },
        })
      })
    )

    render(
      <DeploymentDashboard
        initialDeployments={[
          {
            serviceName: "k8s",
            host: "k8s.gron-studio.com",
            image: "cluster-managed",
            namespace: "gron-services",
            port: 0,
            exposure: "tailscale",
            tailscaleOnly: true,
            dryRun: false,
            commands: [],
            manifest: "",
            containerLogs: "listening",
          },
        ]}
      />
    )

    expect(screen.getByText("k8s")).toBeDefined()
    expect(summaryCardText("Services")).toBe("Services1")
    expect(summaryCardText("Tailscale-only")).toBe("Tailscale-only1")
    expect(summaryCardText("Internet routes")).toBe("Internet routes0")
    expect(screen.queryByText("No deployed containers found")).toBeNull()
    expect(vi.mocked(fetch)).not.toHaveBeenCalled()
  })

  it("treats an empty server-loaded deployment list as final initial data", () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        return new Response(
          JSON.stringify({
            deployments: [
              {
                serviceName: "late-client-fetch",
                host: "late.example.com",
                image: "ship/late:latest",
                namespace: "ship-services",
                port: 3000,
                exposure: "tailscale",
                tailscaleOnly: true,
                dryRun: false,
                commands: [],
                manifest: "",
                containerLogs: "",
              },
            ],
          }),
          { headers: { "Content-Type": "application/json" } }
        )
      })
    )

    render(<DeploymentDashboard initialDeployments={[]} />)

    expect(summaryCardText("Services")).toBe("Services0")
    expect(screen.getByText("No deployed containers found")).toBeDefined()
    expect(screen.queryByText("late-client-fetch")).toBeNull()
    expect(vi.mocked(fetch)).not.toHaveBeenCalled()
  })

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

function summaryCardText(label: string): string {
  const description = screen
    .getAllByText(label)
    .find((element) => element.getAttribute("data-slot") === "card-description")
  const card = description?.closest("[data-slot='card']")
  expect(card).not.toBeNull()
  return card?.textContent ?? ""
}
