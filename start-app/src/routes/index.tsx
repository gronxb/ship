import { createFileRoute } from "@tanstack/react-router"
import { createServerFn } from "@tanstack/react-start"
import { getRequestHost } from "@tanstack/react-start/server"

import { DeploymentDashboard } from "@/components/dashboard/deployment-dashboard"

const loadDeployments = createServerFn().handler(async () => {
  const { readMergedDeployments } = await import("@/lib/deployment-state")
  const { visibleDeployments } = await import("@/lib/deployment-visibility")

  return visibleDeployments(
    await readMergedDeployments(),
    getRequestHost({ xForwardedHost: true })
  )
})

export const Route = createFileRoute("/")({
  component: DashboardRoute,
  loader: () => loadDeployments(),
})

function DashboardRoute() {
  const deployments = Route.useLoaderData()

  return <DeploymentDashboard initialDeployments={deployments} />
}
