import { execFile } from "node:child_process"
import { mkdtemp, rm, writeFile } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join, resolve } from "node:path"
import { promisify } from "node:util"
import { describe, expect, it } from "vitest"
import { z } from "zod"

const execFileAsync = promisify(execFile)
const repoRoot = resolve(process.cwd(), "..")
const shipEnv = { ...process.env, SHIP_CONFIG: "/dev/null" }

const DeploymentResult = z.object({
  serviceName: z.string(),
  host: z.string(),
  image: z.string(),
  namespace: z.string(),
  dockerfilePath: z.string(),
  contextDir: z.string(),
  port: z.number(),
  exposure: z.enum(["tailscale", "internet"]),
  tailscaleOnly: z.boolean(),
  dryRun: z.literal(true),
  commands: z.array(z.string()),
  manifest: z.string(),
})

describe("ship dry-run", () => {
  it("returns a Tailscale-only host route plan when Dockerfile exists", async () => {
    const dir = await mkdtemp(join(tmpdir(), "ship-deploy-"))
    const dockerfile = join(dir, "Dockerfile")
    await writeFile(
      dockerfile,
      'FROM busybox\nCMD ["sh", "-c", "httpd -f -p 8080"]\n'
    )

    try {
      const output = await execFileAsync(
        "go",
        [
          "run",
          "./cmd/ship",
          "--service",
          "demo",
          "--cwd",
          dir,
          "--dry-run",
          "--json",
        ],
        { cwd: repoRoot, env: shipEnv }
      )
      const result = DeploymentResult.parse(JSON.parse(output.stdout))

      expect(result.host).toBe("demo.example.com")
      expect(result.commands).toContain(
        `kind load docker-image --name 'ship' '${result.image}'`
      )
      expect(result.manifest).toContain('ship.local/tailscale-only: "true"')
      expect(result.exposure).toBe("tailscale")
      expect(result.manifest).toContain("hostnames:\n    - demo.example.com")
      expect(result.port).toBe(8080)
    } finally {
      await rm(dir, { recursive: true, force: true })
    }
  })

  it("returns a Cloudflare Tunnel plan when internet exposure is requested", async () => {
    const dir = await mkdtemp(join(tmpdir(), "ship-deploy-"))
    await writeFile(join(dir, "Dockerfile"), "FROM busybox\n")

    try {
      const output = await execFileAsync(
        "go",
        [
          "run",
          "./cmd/ship",
          "--service",
          "demo",
          "--cwd",
          dir,
          "--exposure",
          "internet",
          "--dry-run",
          "--json",
        ],
        { cwd: repoRoot, env: shipEnv }
      )
      const result = DeploymentResult.parse(JSON.parse(output.stdout))

      expect(result.exposure).toBe("internet")
      expect(result.tailscaleOnly).toBe(false)
      expect(result.manifest).toContain('ship.local/exposure: "internet"')
      expect(result.manifest).not.toContain("kind: Ingress")
      expect(result.manifest).not.toContain('tailscale.com/funnel: "true"')
      expect(result.commands.join("\n")).toContain(
        "cloudflare tunnel expose demo.example.com"
      )
    } finally {
      await rm(dir, { recursive: true, force: true })
    }
  })

  it("rejects invalid service names before Dockerfile lookup", async () => {
    await expect(
      execFileAsync(
        "go",
        [
          "run",
          "./cmd/ship",
          "--service",
          "Bad_Name",
          "--cwd",
          "/tmp/missing-Dockerfile",
          "--dry-run",
        ],
        { cwd: repoRoot, env: shipEnv }
      )
    ).rejects.toMatchObject({
      stderr: expect.stringContaining("service name must be DNS-safe"),
    })
  })
})
