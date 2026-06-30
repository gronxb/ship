import { createFileRoute } from "@tanstack/react-router"

import {
  changeDeploymentExposure,
  listDeployments,
} from "@/lib/deployment-handlers"

export const Route = createFileRoute("/api/deployments")({
  server: {
    handlers: {
      GET: listDeployments,
      PATCH: ({ request }) => changeDeploymentExposure(request),
    },
  },
})
