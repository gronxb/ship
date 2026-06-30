import { createFileRoute } from "@tanstack/react-router"

import { DeploymentDashboard } from "@/components/dashboard/deployment-dashboard"

export const Route = createFileRoute("/")({ component: DeploymentDashboard })
