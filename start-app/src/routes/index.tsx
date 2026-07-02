import { createFileRoute } from "@tanstack/react-router"
import { createServerFn } from "@tanstack/react-start"

import { DeploymentDashboard } from "@/components/dashboard/deployment-dashboard"

const loadDeployments = createServerFn().handler(async () => {
  const { readMergedDeployments } = await import("@/lib/deployment-state")

  return readMergedDeployments()
})

export const Route = createFileRoute("/")({
  component: DashboardRoute,
  loader: () => loadDeployments(),
})

function DashboardRoute() {
  const deployments = Route.useLoaderData()

  return <DeploymentDashboard initialDeployments={deployments} />
}
