import {
  domain,
  internetGatewayName,
  serviceNamespace,
} from "./deployment-config"
import { execFileAsync } from "./deployment-command"
import { HttpRouteList, IngressList } from "./deployment-schema"
import type {
  DeploymentResult,
  Exposure,
  HttpRoute,
  Ingress,
} from "./deployment-schema"

export async function readClusterDeployments(): Promise<
  readonly DeploymentResult[]
> {
  const routes = await readHttpRoutes()
  const ingresses = await readInternetIngresses()
  const ingressByService = new Map(
    ingresses.map((item) => [ingressServiceName(item), item])
  )
  return Promise.all(
    routes.map((item) =>
      clusterDeployment(item, ingressByService.get(item.metadata.name))
    )
  )
}

async function readHttpRoutes(): Promise<readonly HttpRoute[]> {
  try {
    const output = await execFileAsync("kubectl", [
      "get",
      "httproute",
      "-n",
      serviceNamespace,
      "-o",
      "json",
    ])
    const routes = HttpRouteList.parse(JSON.parse(output.stdout))
    return routes.items
  } catch (error) {
    if (error instanceof SyntaxError) {
      return []
    }
    if (error instanceof Error && "stderr" in error) {
      return []
    }
    throw error
  }
}

async function readInternetIngresses(): Promise<readonly Ingress[]> {
  try {
    const output = await execFileAsync("kubectl", [
      "get",
      "ingress",
      "-n",
      serviceNamespace,
      "-l",
      "ship.local/exposure=internet",
      "-o",
      "json",
    ])
    const ingresses = IngressList.parse(JSON.parse(output.stdout))
    return ingresses.items
  } catch (error) {
    if (error instanceof SyntaxError) {
      return []
    }
    if (error instanceof Error && "stderr" in error) {
      return []
    }
    throw error
  }
}

async function clusterDeployment(
  item: HttpRoute,
  internetIngress: Ingress | undefined
): Promise<DeploymentResult> {
  const serviceName = item.metadata.name
  const exposure = internetIngress ? "internet" : exposureForRoute(item)
  const routeHost = item.spec.hostnames.at(0)
  return {
    serviceName,
    host:
      ingressAddress(internetIngress) ??
      routeHost ??
      `${serviceName}.${domain}`,
    image: "cluster-managed",
    namespace: item.metadata.namespace,
    dockerfilePath: "",
    contextDir: "",
    port: 0,
    exposure,
    tailscaleOnly: exposure === "tailscale",
    dryRun: false,
    commands: [],
    manifest: "",
    containerLogs: await readContainerLogs(
      serviceName,
      item.metadata.namespace
    ),
  }
}

function ingressServiceName(item: Ingress): string {
  return item.metadata.labels?.["app.kubernetes.io/name"] ?? item.metadata.name
}

function ingressAddress(item: Ingress | undefined): string | undefined {
  const address = item?.status.loadBalancer.ingress[0]
  return address?.hostname ?? address?.ip
}

function exposureForRoute(item: HttpRoute): Exposure {
  const label = item.metadata.labels?.["ship.local/exposure"]
  if (label === "internet" || label === "tailscale") {
    return label
  }
  return item.spec.parentRefs.some((ref) => ref.name === internetGatewayName)
    ? "internet"
    : "tailscale"
}

async function readContainerLogs(
  serviceName: string,
  namespace: string
): Promise<string> {
  try {
    const output = await execFileAsync("kubectl", [
      "logs",
      `deployment/${serviceName}`,
      "-n",
      namespace,
      "--tail=80",
      "--all-containers=true",
    ])
    return output.stdout.trim()
  } catch (error) {
    if (error instanceof Error) {
      return `Container logs unavailable: ${error.message}`
    }
    throw error
  }
}
