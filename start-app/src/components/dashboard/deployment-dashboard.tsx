import { useEffect, useMemo, useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"

import { DeploymentCard } from "./deployment-card"
import { LogPanel } from "./log-panel"
import type { Deployment, DeploymentsResponse } from "./types"
import { DeploymentsResponse as DeploymentsResponseSchema } from "./types"

export function DeploymentDashboard() {
  const [deployments, setDeployments] = useState<readonly Deployment[]>([])
  const [requestLog, setRequestLog] = useState<readonly string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  async function loadDeployments(): Promise<void> {
    setRequestLog((current) => ["GET /api/deployments", ...current].slice(0, 8))
    const response = await fetch("/api/deployments")
    const data: DeploymentsResponse = DeploymentsResponseSchema.parse(
      await response.json()
    )
    setDeployments(data.deployments)
  }

  async function refreshDeployments(): Promise<void> {
    setLoading(true)
    setError("")
    try {
      await loadDeployments()
    } catch (caught) {
      if (caught instanceof Error) {
        setError(caught.message)
        return
      }
      throw caught
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refreshDeployments().catch((caught: unknown) => {
      if (caught instanceof Error) {
        setError(caught.message)
        setLoading(false)
        return
      }
      throw caught
    })
  }, [])

  const summary = useMemo(() => deploymentSummary(deployments), [deployments])

  return (
    <main className="min-h-svh bg-background px-4 py-8 text-foreground md:px-8">
      <div className="mx-auto grid w-full max-w-6xl gap-6">
        <header className="flex flex-col gap-5 md:flex-row md:items-end md:justify-between">
          <div>
            <Badge variant="outline">Ship Gateway</Badge>
            <h1 className="mt-4 max-w-3xl text-4xl font-semibold tracking-normal text-balance md:text-5xl">
              Deployed containers
            </h1>
            <p className="mt-4 max-w-2xl text-base leading-7 text-muted-foreground">
              Review running services, route exposure, network requests, and
              terminal output from the Kubernetes edge.
            </p>
          </div>
          <Button disabled={loading} onClick={refreshDeployments} type="button">
            {loading ? "Refreshing" : "Refresh"}
          </Button>
        </header>

        <section className="grid gap-3 md:grid-cols-3">
          <SummaryCard label="Services" value={String(summary.total)} />
          <SummaryCard label="Tailscale-only" value={String(summary.tailscale)} />
          <SummaryCard label="Internet routes" value={String(summary.internet)} />
        </section>

        {error ? (
          <Card className="border-destructive/30 bg-destructive/5">
            <CardHeader>
              <CardTitle>Dashboard request failed</CardTitle>
              <CardDescription>{error}</CardDescription>
            </CardHeader>
          </Card>
        ) : null}

        <section className="grid gap-4">
          {loading ? (
            <LoadingCards />
          ) : deployments.length === 0 ? (
            <EmptyState requestLog={requestLog} />
          ) : (
            deployments.map((deployment) => (
              <DeploymentCard
                deployment={deployment}
                key={deployment.host}
                requestLog={requestLog}
              />
            ))
          )}
        </section>
      </div>
    </main>
  )
}

function SummaryCard({
  label,
  value,
}: {
  readonly label: string
  readonly value: string
}) {
  return (
    <Card size="sm">
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle className="text-3xl tabular-nums">{value}</CardTitle>
      </CardHeader>
    </Card>
  )
}

function EmptyState({ requestLog }: { readonly requestLog: readonly string[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>No deployed containers found</CardTitle>
        <CardDescription>
          The dashboard is read-only. Deploy services from the CLI, then refresh
          this page to inspect cards and logs.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <LogPanel
          lines={
            requestLog.length > 0
              ? requestLog
              : ["GET /api/deployments", "No deployments returned."]
          }
        />
      </CardContent>
    </Card>
  )
}

function LoadingCards() {
  return (
    <>
      <Skeleton className="h-56" />
      <Skeleton className="h-56" />
    </>
  )
}

function deploymentSummary(deployments: readonly Deployment[]) {
  return {
    total: deployments.length,
    tailscale: deployments.filter((deployment) => deployment.tailscaleOnly)
      .length,
    internet: deployments.filter((deployment) => !deployment.tailscaleOnly)
      .length,
  }
}
