import { mkdir, readFile, writeFile } from "node:fs/promises"
import { dirname } from "node:path"
import { z } from "zod"

import { statePath } from "./deployment-config"
import { readClusterDeployments } from "./deployment-cluster"
import { DeploymentResult } from "./deployment-schema"
import type { DeploymentResult as DeploymentResultType } from "./deployment-schema"

export async function readMergedDeployments(): Promise<
  readonly DeploymentResultType[]
> {
  const saved = await readDeployments()
  const cluster = await readClusterDeployments()
  const clusterNames = new Set(cluster.map((item) => item.serviceName))
  return [
    ...cluster,
    ...saved.filter((item) => !clusterNames.has(item.serviceName)),
  ]
}

export async function saveDeployment(
  item: DeploymentResultType
): Promise<void> {
  const current = await readDeployments()
  const next = [
    item,
    ...current.filter((row) => row.serviceName !== item.serviceName),
  ].slice(0, 25)
  await mkdir(dirname(statePath), { recursive: true })
  await writeFile(statePath, JSON.stringify(next, null, 2))
}

async function readDeployments(): Promise<readonly DeploymentResultType[]> {
  try {
    const raw = await readFile(statePath, "utf8")
    return z.array(DeploymentResult).parse(JSON.parse(raw))
  } catch (error) {
    if (error instanceof z.ZodError || error instanceof SyntaxError) {
      return []
    }
    if (error instanceof Error && "code" in error && error.code === "ENOENT") {
      return []
    }
    throw error
  }
}
