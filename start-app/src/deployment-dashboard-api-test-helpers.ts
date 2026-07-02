import { vi } from "vitest"

import type * as DeploymentCommand from "./lib/deployment-command"
import type * as DeploymentDNS from "./lib/deployment-dns"
import type * as DeploymentState from "./lib/deployment-state"
import type { DeploymentResult } from "./lib/deployment-schema"

vi.mock("./lib/deployment-command", () => ({
  execFileAsync: vi.fn(),
  execFileWithInput: vi.fn(),
}))

vi.mock("./lib/deployment-dns", () => ({
  resolveIPv4: vi.fn(),
}))

vi.mock("./lib/deployment-state", async (importOriginal) => {
  const actual = await importOriginal<typeof DeploymentState>()
  return {
    ...actual,
    readMergedDeployments: vi.fn(),
  }
})

export async function mockedDeploymentModules(): Promise<{
  readonly execFileAsync: ReturnType<
    typeof vi.mocked<typeof DeploymentCommand.execFileAsync>
  >
  readonly execFileWithInput: ReturnType<
    typeof vi.mocked<typeof DeploymentCommand.execFileWithInput>
  >
  readonly resolveIPv4: ReturnType<
    typeof vi.mocked<typeof DeploymentDNS.resolveIPv4>
  >
  readonly readMergedDeployments: ReturnType<
    typeof vi.mocked<typeof DeploymentState.readMergedDeployments>
  >
}> {
  const command = await import("./lib/deployment-command")
  const dns = await import("./lib/deployment-dns")
  const state = await import("./lib/deployment-state")
  return {
    execFileAsync: vi.mocked(command.execFileAsync),
    execFileWithInput: vi.mocked(command.execFileWithInput),
    resolveIPv4: vi.mocked(dns.resolveIPv4),
    readMergedDeployments: vi.mocked(state.readMergedDeployments),
  }
}

export function exposureRequest(body: unknown): Request {
  return new Request("http://ship.local/api/deployments", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
}

export function deploymentFixture({
  namespace,
  serviceName,
  tailscaleOnly,
}: {
  readonly namespace: string
  readonly serviceName: string
  readonly tailscaleOnly: boolean
}): DeploymentResult {
  return {
    serviceName,
    host: `${serviceName}.example.com`,
    image: "cluster-managed",
    namespace,
    dockerfilePath: "",
    contextDir: "",
    port: 0,
    exposure: tailscaleOnly ? "tailscale" : "internet",
    tailscaleOnly,
    dryRun: false,
    commands: [],
    manifest: "",
    containerLogs: "",
  }
}
