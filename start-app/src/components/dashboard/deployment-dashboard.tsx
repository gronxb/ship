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

export function DeploymentDashboard({
  initialDeployments,
}: {
  readonly initialDeployments?: readonly Deployment[]
}) {
  const hasInitialDeployments = initialDeployments !== undefined
  const [deployments, setDeployments] = useState<readonly Deployment[]>(
    initialDeployments ?? []
  )
  const [requestLog, setRequestLog] = useState<readonly string[]>([])
  const [loading, setLoading] = useState(!hasInitialDeployments)
  const [error, setError] = useState("")

  function recordRequest(request: string): void {
    setRequestLog((current) => [request, ...current].slice(0, 8))
  }

  async function loadDeployments(): Promise<void> {
    recordRequest("GET /api/deployments")
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
    if (hasInitialDeployments) {
      return
    }
    refreshDeployments().catch((caught: unknown) => {
      if (caught instanceof Error) {
        setError(caught.message)
        setLoading(false)
        return
      }
      throw caught
    })
  }, [hasInitialDeployments])

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
          <div className="flex flex-wrap gap-2">
            <Button asChild variant="outline">
              <a
                aria-label="Star on GitHub"
                href="https://github.com/gronxb/ship"
                rel="noreferrer"
                target="_blank"
              >
                <GitHubMark />
                Star on GitHub
              </a>
            </Button>
            <Button
              disabled={loading}
              onClick={refreshDeployments}
              type="button"
            >
              {loading ? "Refreshing" : "Refresh"}
            </Button>
          </div>
        </header>

        <section className="grid gap-3 md:grid-cols-3">
          <SummaryCard label="Services" value={String(summary.total)} />
          <SummaryCard
            label="Tailscale-only"
            value={String(summary.tailscale)}
          />
          <SummaryCard
            label="Internet routes"
            value={String(summary.internet)}
          />
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
                onExposureChanged={refreshDeployments}
                onRequest={recordRequest}
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

function EmptyState({
  requestLog,
}: {
  readonly requestLog: readonly string[]
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>No deployed containers found</CardTitle>
        <CardDescription>
          Deploy services from the CLI, then refresh this page to inspect cards,
          logs, and exposure controls.
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

function GitHubMark() {
  return (
    <svg aria-hidden="true" fill="currentColor" viewBox="0 0 24 24">
      <path d="M12 .5a12 12 0 0 0-3.8 23.39c.6.11.82-.26.82-.58v-2.1c-3.34.73-4.04-1.42-4.04-1.42-.55-1.39-1.34-1.76-1.34-1.76-1.09-.75.08-.73.08-.73 1.21.08 1.85 1.24 1.85 1.24 1.07 1.84 2.82 1.31 3.5 1 .11-.78.42-1.31.76-1.61-2.66-.3-5.46-1.33-5.46-5.93 0-1.31.47-2.38 1.24-3.22-.12-.3-.54-1.52.12-3.18 0 0 1.01-.32 3.3 1.23A11.46 11.46 0 0 1 12 5.72c1.02 0 2.04.14 3 .41 2.29-1.55 3.3-1.23 3.3-1.23.66 1.66.24 2.88.12 3.18.77.84 1.23 1.91 1.23 3.22 0 4.61-2.8 5.63-5.48 5.93.43.37.82 1.1.82 2.22v3.29c0 .32.22.69.83.57A12 12 0 0 0 12 .5Z" />
    </svg>
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
