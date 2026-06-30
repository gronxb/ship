import { useEffect, useState } from "react"

import { Badge } from "@/components/ui/badge"
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

export function DeploymentCard({
  deployment,
  requestLog,
}: {
  readonly deployment: Deployment
  readonly requestLog: readonly string[]
}) {
  const [activeTab, setActiveTab] = useState<DashboardTab>("network")

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

  return (
    <Card className="transition-colors hover:ring-foreground/20">
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center gap-2">
          <span>{deployment.serviceName}</span>
          <ExposureBadge deployment={deployment} />
          {deployment.dryRun ? (
            <Badge variant="secondary">dry-run</Badge>
          ) : (
            <Badge variant="outline">applied</Badge>
          )}
        </CardTitle>
        <CardDescription className="break-all">{deployment.host}</CardDescription>
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
            <LogPanel
              lines={[
                `GET https://${deployment.host}`,
                "GET /api/deployments",
                ...requestLog,
              ]}
            />
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
      <p className="mt-1 break-all font-medium">{value}</p>
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
