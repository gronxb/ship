import type { DeploymentResult } from "./deployment-schema"
import { dashboardHost as configuredDashboardHost } from "./deployment-config"

export function visibleDeployments(
  deployments: readonly DeploymentResult[],
  dashboardHost: string | undefined
): readonly DeploymentResult[] {
  if (normalizedHost(dashboardHost) === "") {
    return deployments
  }
  return deployments.filter(
    (deployment) => !isDashboardDeployment(deployment, dashboardHost)
  )
}

export function isDashboardDeployment(
  deployment: DeploymentResult,
  requestHost: string | undefined
): boolean {
  const currentHost = normalizedHost(requestHost)
  return currentHost !== "" && normalizedHost(deployment.host) === currentHost
}

export function dashboardHostFromRequest(request: Request): string {
  const forwardedHost = request.headers.get("x-forwarded-host")
  if (forwardedHost !== null && forwardedHost.trim() !== "") {
    return forwardedHost
  }
  if (configuredDashboardHost !== "") {
    return configuredDashboardHost
  }
  return new URL(request.url).host
}

function normalizedHost(value: string | undefined): string {
  if (value === undefined) {
    return ""
  }
  const [forwardedHost = ""] = value.trim().split(",")
  const host = forwardedHost.trim().toLowerCase()
  if (host === "") {
    return ""
  }

  try {
    return new URL(host.includes("://") ? host : `https://${host}`).hostname
  } catch (caught) {
    if (caught instanceof TypeError) {
      return host
    }
    throw caught
  }
}
