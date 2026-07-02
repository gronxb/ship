import { useEffect, useState } from "react"
import { z } from "zod"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

import { DetailsPanel } from "./deployment-details-panel"
import {
  deploymentUrl,
  exposureWaitEstimate,
  exposureWaitReason,
  exposureStatus,
  PreviewPanel,
} from "./deployment-route-panel"
import type { ExposureUpdate } from "./deployment-route-panel"
import { LogPanel } from "./log-panel"
import type { Deployment, DeploymentExposureUpdate, Exposure } from "./types"
import { DeploymentExposureResponse } from "./types"

type DashboardTab = "network" | "terminal" | "details"

const deploymentApiError = z.object({ error: z.string() })

export function DeploymentCard({
  deployment,
  onExposureChanged,
}: {
  readonly deployment: Deployment
  readonly onExposureChanged: (
    update: DeploymentExposureUpdate
  ) => Promise<void>
}) {
  const [activeTab, setActiveTab] = useState<DashboardTab>("network")
  const [updatingExposure, setUpdatingExposure] =
    useState<ExposureUpdate | null>(null)
  const [exposureError, setExposureError] = useState("")
  const nextExposure = updatingExposure?.exposure ?? exposureTarget(deployment)

  useEffect(() => {
    function syncHashTab(): void {
      setActiveTab(tabFromHash())
    }
    syncHashTab()
    window.addEventListener("hashchange", syncHashTab)
    return () => {
      window.removeEventListener("hashchange", syncHashTab)
    }
  }, [])

  async function changeExposure(
    exposure: Deployment["exposure"]
  ): Promise<void> {
    setUpdatingExposure({ exposure })
    setExposureError("")
    try {
      const response = await fetch("/api/deployments", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          serviceName: deployment.serviceName,
          namespace: deployment.namespace,
          exposure,
        }),
      })
      if (!response.ok) {
        throw new Error(await responseErrorMessage(response))
      }
      const data = DeploymentExposureResponse.parse(await response.json())
      await onExposureChanged(data.deployment)
    } catch (caught) {
      if (caught instanceof Error) {
        setExposureError(caught.message)
        return
      }
      throw caught
    } finally {
      setUpdatingExposure(null)
    }
  }

  return (
    <Card
      className={cn(
        "group transition-colors hover:bg-card/90 hover:ring-foreground/20",
        !deployment.tailscaleOnly &&
          "relative isolate bg-[linear-gradient(140deg,hsl(192_92%_12%/.95),hsl(160_42%_9%/.98)_48%,hsl(42_74%_13%/.9))] shadow-[0_0_0_1px_hsl(189_94%_70%/.16),0_24px_80px_-48px_hsl(189_94%_70%/.9)] ring-cyan-300/55 before:pointer-events-none before:absolute before:inset-0 before:rounded-xl before:border before:border-cyan-200/35 before:[box-shadow:inset_0_1px_0_hsl(45_96%_72%/.28)] hover:ring-cyan-200/75"
      )}
    >
      <CardHeader className="gap-3 md:grid-cols-[1fr_auto]">
        <CardTitle className="flex flex-wrap items-center gap-2">
          <span>{deployment.serviceName}</span>
          <ExposureBadge deployment={deployment} />
          {deployment.dryRun ? (
            <Badge variant="secondary">dry-run</Badge>
          ) : (
            <Badge variant="outline">applied</Badge>
          )}
        </CardTitle>
        <CardDescription className="break-all">
          {deployment.host}
        </CardDescription>
        {!deployment.tailscaleOnly ? (
          <div className="flex flex-wrap items-center gap-2 text-xs text-cyan-100/80">
            <span className="rounded border border-cyan-200/30 bg-cyan-200/10 px-2 py-1 font-medium text-cyan-50">
              Internet network
            </span>
            <span>global ship route</span>
          </div>
        ) : null}
        <div className="flex flex-wrap gap-2 md:col-start-2 md:row-span-2 md:row-start-1 md:justify-end">
          <Button asChild size="sm" variant="outline">
            <a
              href={deploymentUrl(deployment)}
              rel="noreferrer"
              target="_blank"
            >
              Open URL
            </a>
          </Button>
          <Button
            aria-pressed={false}
            disabled={updatingExposure !== null}
            onClick={() => void changeExposure(nextExposure)}
            size="sm"
            type="button"
            variant="outline"
          >
            {exposureButtonLabel(nextExposure, updatingExposure)}
          </Button>
          {exposureError ? (
            <p className="basis-full text-right text-xs text-destructive">
              {exposureError}
            </p>
          ) : null}
        </div>
        {updatingExposure !== null ? (
          <ExposureWaitNotice
            deployment={deployment}
            update={updatingExposure}
          />
        ) : null}
      </CardHeader>
      <CardContent className="grid gap-4">
        <PreviewPanel deployment={deployment} />
        <Tabs
          onValueChange={(value) => {
            if (isDashboardTab(value)) {
              setActiveTab(value)
            }
          }}
          value={activeTab}
        >
          <TabsList className="w-full justify-start overflow-x-auto">
            <TabsTrigger className="min-w-max" value="network">
              <span className="sm:hidden">Network</span>
              <span className="hidden sm:inline">Network requests</span>
            </TabsTrigger>
            <TabsTrigger className="min-w-max" value="terminal">
              <span className="sm:hidden">Terminal</span>
              <span className="hidden sm:inline">Terminal logs</span>
            </TabsTrigger>
            <TabsTrigger className="min-w-max" value="details">
              Details
            </TabsTrigger>
          </TabsList>
          <TabsContent value="network">
            <LogPanel lines={[`GET https://${deployment.host}`]} />
          </TabsContent>
          <TabsContent value="terminal">
            <LogPanel
              lines={[
                ...deployment.commands,
                deployment.containerLogs || "No container logs recorded.",
              ]}
            />
          </TabsContent>
          <TabsContent value="details">
            <DetailsPanel deployment={deployment} />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

function ExposureWaitNotice({
  deployment,
  update,
}: {
  readonly deployment: Deployment
  readonly update: ExposureUpdate
}) {
  return (
    <div
      className="grid gap-2 rounded-md border border-border/80 bg-muted/35 px-3 py-2 text-xs leading-5 text-muted-foreground md:col-span-2"
      role="status"
    >
      <div className="flex items-start gap-2">
        <span
          aria-hidden="true"
          className="mt-1 size-2 shrink-0 rounded-full bg-emerald-500 shadow-[0_0_0_4px_hsl(142_76%_36%/.12)]"
        />
        <div className="grid gap-1">
          <p className="font-medium text-foreground">
            {exposureStatus(deployment, update)}
          </p>
          <p>{exposureWaitReason(update)}</p>
          <p>{exposureWaitEstimate(update)}</p>
        </div>
      </div>
    </div>
  )
}

function exposureTarget(deployment: Deployment): Exposure {
  return deployment.tailscaleOnly ? "internet" : "tailscale"
}

function exposureButtonLabel(
  exposure: Exposure,
  update: ExposureUpdate | null
): string {
  if (update !== null) {
    return "Waiting"
  }
  return exposure === "internet" ? "Expose to internet" : "Use Tailscale only"
}

async function responseErrorMessage(response: Response): Promise<string> {
  let body: unknown
  try {
    body = await response.json()
  } catch (caught) {
    if (caught instanceof SyntaxError) {
      return response.statusText || "Request failed"
    }
    throw caught
  }
  const parsed = deploymentApiError.safeParse(body)
  return parsed.success
    ? parsed.data.error
    : response.statusText || "Request failed"
}

function ExposureBadge({ deployment }: { readonly deployment: Deployment }) {
  return deployment.tailscaleOnly ? (
    <Badge className="border-emerald-700/30 text-emerald-700" variant="outline">
      Tailscale-only
    </Badge>
  ) : (
    <Badge
      className="border-cyan-200/40 bg-cyan-200/10 text-cyan-50"
      variant="outline"
    >
      Internet
    </Badge>
  )
}

function tabFromHash(): DashboardTab {
  const params = new URLSearchParams(window.location.search)
  const queryTab = params.get("tab")
  if (queryTab && isDashboardTab(queryTab)) {
    return queryTab
  }
  const tab = window.location.hash.slice(1)
  return isDashboardTab(tab) ? tab : "network"
}

function isDashboardTab(value: string): value is DashboardTab {
  return value === "network" || value === "terminal" || value === "details"
}
