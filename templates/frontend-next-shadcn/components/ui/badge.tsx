import { cn } from "@/lib/utils"

const colors: Record<string, string> = {
  active: "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  paid: "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  pending: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300",
  open: "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-300",
  blocked: "border-danger/30 bg-danger/10 text-danger",
  failed: "border-danger/30 bg-danger/10 text-danger",
  refunded: "border-zinc-500/30 bg-zinc-500/10 text-zinc-700 dark:text-zinc-300",
  past_due: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300"
}

export function Badge({ value }: { value: unknown }) {
  const text = String(value ?? "")
  return (
    <span className={cn("inline-flex rounded-full border px-2 py-0.5 text-xs font-medium", colors[text] ?? "border-border bg-muted")}>
      {text.replaceAll("_", " ")}
    </span>
  )
}
