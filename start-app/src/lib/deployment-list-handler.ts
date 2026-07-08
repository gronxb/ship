import { readMergedDeployments } from "./deployment-state"
import {
  dashboardHostFromRequest,
  visibleDeployments,
} from "./deployment-visibility"

export async function listDeployments(request: Request): Promise<Response> {
  return Response.json({
    deployments: visibleDeployments(
      await readMergedDeployments(),
      dashboardHostFromRequest(request)
    ),
  })
}
