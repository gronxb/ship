import type { Deployment } from "./types"

export type ExposureUpdate = {
  readonly exposure: Deployment["exposure"]
}

export function PreviewPanel({
  deployment,
}: {
  readonly deployment: Deployment
}) {
  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <div className="flex items-center justify-between border-b bg-muted/40 px-3 py-2">
        <span className="truncate text-xs text-muted-foreground">
          {deploymentUrl(deployment)}
        </span>
        <span className="rounded border px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
          PREVIEW
        </span>
      </div>
      <iframe
        className="h-56 w-full bg-background"
        loading="lazy"
        referrerPolicy="no-referrer"
        sandbox="allow-forms allow-popups allow-same-origin allow-scripts"
        src={deploymentUrl(deployment)}
        title={`Preview of ${deployment.serviceName}`}
      />
    </div>
  )
}

export function deploymentUrl(deployment: Deployment): string {
  return `https://${deployment.host}`
}

export function exposureStatus(
  deployment: Deployment,
  update: ExposureUpdate
): string {
  return update.exposure === "internet"
    ? `Waiting for DNS and route propagation for ${deployment.host}...`
    : `Waiting for Tailscale route propagation for ${deployment.host}...`
}

export function exposureWaitReason(update: ExposureUpdate): string {
  return update.exposure === "internet"
    ? "Ship is waiting for public DNS caches to leave the Tailscale address, then probing the internet route before it reports success."
    : "Ship is waiting for DNS caches to leave Cloudflare, then probing the Tailscale Gateway before it reports success."
}

export function exposureWaitEstimate(update: ExposureUpdate): string {
  return update.exposure === "internet"
    ? "This usually takes about 5 minutes because Cloudflare proxied records use a 300 second cache window."
    : "This usually takes 5 to 7 minutes while recursive DNS and the local route cache converge."
}
