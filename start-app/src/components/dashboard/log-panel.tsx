import { ScrollArea } from "@/components/ui/scroll-area"

export function LogPanel({ lines }: { readonly lines: readonly string[] }) {
  return (
    <ScrollArea className="h-64 rounded-lg border bg-zinc-950">
      <pre className="p-4 font-mono text-xs leading-6 whitespace-pre-wrap text-zinc-100">
        {lines.filter(Boolean).join("\n")}
      </pre>
    </ScrollArea>
  )
}
