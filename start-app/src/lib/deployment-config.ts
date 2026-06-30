import { existsSync, readFileSync } from "node:fs"
import { join } from "node:path"

export const statePath = join(
  process.cwd(),
  "..",
  ".omo",
  "ulw-dashboard",
  "deployments.json"
)

const shipConfig = loadShipConfig()

export const serviceNamespace = configValue("SHIP_NAMESPACE", "ship-services")
export const internetGatewayName = configValue(
  "SHIP_INTERNET_GATEWAY_NAME",
  "ship-internet"
)
export const domain = configValue("SHIP_DOMAIN", "example.com")

function configValue(name: string, fallback: string): string {
  return process.env[name] || shipConfig[name] || fallback
}

function loadShipConfig(): Readonly<Record<string, string>> {
  const path =
    process.env.SHIP_CONFIG ||
    join(
      process.env.XDG_CONFIG_HOME || join(process.env.HOME || "", ".config"),
      "ship",
      "config.env"
    )
  if (!path || !existsSync(path)) {
    return {}
  }
  return Object.fromEntries(
    readFileSync(path, "utf8")
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line !== "" && !line.startsWith("#"))
      .map((line) => {
        const [key, ...rest] = line.split("=")
        return [
          key.trim(),
          rest
            .join("=")
            .trim()
            .replace(/^['"]|['"]$/g, ""),
        ]
      })
  )
}
