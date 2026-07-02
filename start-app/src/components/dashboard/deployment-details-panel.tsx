import { Separator } from "@/components/ui/separator"

import { LogPanel } from "./log-panel"
import type { Deployment } from "./types"

export function DetailsPanel({
  deployment,
}: {
  readonly deployment: Deployment
}) {
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
