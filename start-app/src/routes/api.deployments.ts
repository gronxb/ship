import { createFileRoute } from "@tanstack/react-router"

import { changeDeploymentExposure } from "@/lib/deployment-handlers"
import { listDeployments } from "@/lib/deployment-list-handler"

export const Route = createFileRoute("/api/deployments")({
  server: {
    handlers: {
      GET: ({ request }) => listDeployments(request),
      PATCH: ({ request }) => changeDeploymentExposure(request),
    },
  },
})
