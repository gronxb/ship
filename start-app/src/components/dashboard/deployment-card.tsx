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
import { Separator } from "@/components/ui/separator"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

import { LogPanel } from "./log-panel"
import type { Deployment } from "./types"

type DashboardTab = "network" | "terminal" | "details"

const deploymentApiError = z.object({ error: z.string() })

export function DeploymentCard({
  deployment,
  onExposureChanged,
}: {
  readonly deployment: Deployment
  readonly onExposureChanged: () => Promise<void>
}) {
  const [activeTab, setActiveTab] = useState<DashboardTab>("network")
  const [updatingExposure, setUpdatingExposure] = useState(false)
  const [exposureError, setExposureError] = useState("")

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

  async function exposeToInternet(): Promise<void> {
    setUpdatingExposure(true)
    setExposureError("")
    try {
      const response = await fetch("/api/deployments", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          serviceName: deployment.serviceName,
          namespace: deployment.namespace,
          exposure: "internet",
        }),
      })
      if (!response.ok) {
        throw new Error(await responseErrorMessage(response))
      }
      await onExposureChanged()
    } catch (caught) {
      if (caught instanceof Error) {
        setExposureError(caught.message)
        return
      }
      throw caught
    } finally {
      setUpdatingExposure(false)
    }
  }

  return (
    <Card className="transition-colors hover:ring-foreground/20">
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
        <div className="grid gap-2 md:col-start-2 md:row-span-2 md:row-start-1 md:justify-items-end">
          {deployment.tailscaleOnly ? (
            <Button
              aria-pressed={false}
              disabled={updatingExposure}
              onClick={() => void exposeToInternet()}
              size="sm"
              type="button"
              variant="outline"
            >
              {updatingExposure ? "Exposing" : "Expose to internet"}
            </Button>
          ) : null}
          {exposureError ? (
            <p className="max-w-64 text-right text-xs text-destructive">
              {exposureError}
            </p>
          ) : null}
        </div>
      </CardHeader>
      <CardContent>
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
    <Badge className="border-blue-700/30 text-blue-700" variant="outline">
      Internet
    </Badge>
  )
}

function DetailsPanel({ deployment }: { readonly deployment: Deployment }) {
  return (
    <div className="grid gap-4 rounded-lg border bg-muted/40 p-4 text-sm">
      <div className="grid gap-3 md:grid-cols-2">
        <KeyValue label="Image" value={deployment.image} />
        <KeyValue label="Namespace" value={deployment.namespace} />
        <KeyValue
          label="Port"
          value={deployment.port > 0 ? String(deployment.port) : "not reported"}
        />
        <KeyValue
          label="Recorded"
          value={deployment.createdAt ?? "cluster discovery"}
        />
      </div>
      {deployment.manifest ? (
        <>
          <Separator />
          <LogPanel lines={[deployment.manifest]} />
        </>
      ) : null}
    </div>
  )
}

function KeyValue({
  label,
  value,
}: {
  readonly label: string
  readonly value: string
}) {
  return (
    <div>
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-1 font-medium break-all">{value}</p>
    </div>
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
