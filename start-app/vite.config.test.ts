import { afterEach, describe, expect, it, vi } from "vitest"
import { dashboardAllowedHosts } from "./vite.config"

describe("dashboard host allowlist", () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it("allows the onboarded k8s host for the configured Ship domain", async () => {
    vi.stubEnv("SHIP_DOMAIN", "example.com")

    expect(dashboardAllowedHosts()).toEqual(["k8s.example.com"])
  })

  it("lets an explicit dashboard host override the Ship domain", async () => {
    vi.stubEnv("SHIP_DOMAIN", "example.com")
    vi.stubEnv("SHIP_DASHBOARD_HOST", "dashboard.internal.test")

    expect(dashboardAllowedHosts()).toEqual(["dashboard.internal.test"])
  })
})
