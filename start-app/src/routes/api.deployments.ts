import { createFileRoute } from "@tanstack/react-router"

import { listDeployments } from "@/lib/deployment-handlers"

export const Route = createFileRoute("/api/deployments")({
  server: {
    handlers: {
      GET: listDeployments,
    },
  },
})
