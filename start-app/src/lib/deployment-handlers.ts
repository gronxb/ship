import { readMergedDeployments } from "./deployment-state"

export async function listDeployments(): Promise<Response> {
  return Response.json({ deployments: await readMergedDeployments() })
}
