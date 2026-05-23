"use client"

import { useQuery } from "@tanstack/react-query"
import { Database, Filter } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api"
import { formatDate } from "@/lib/utils"

export default function AuditPage() {
  const audit = useQuery({ queryKey: ["audit"], queryFn: async () => (await api.audit()).data ?? [] })

  return (
    <div className="grid gap-5">
      <div>
        <h1 className="text-2xl font-semibold">Audit log</h1>
        <p className="mt-1 text-sm text-foreground/58">Immutable activity across auth, resources, exports, files, and custom actions.</p>
      </div>
      <section className="rounded-lg border border-border bg-panel shadow-panel">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
          <Input className="max-w-xs" placeholder="Filter by actor or resource" />
          <Button variant="outline" size="sm">
            <Filter className="h-4 w-4" />
            Filters
          </Button>
        </div>
        <div className="divide-y divide-border">
          {audit.data?.map((event) => (
            <div key={String(event.id)} className="grid gap-2 px-4 py-3 md:grid-cols-[220px_1fr_180px]">
              <div className="text-sm font-medium">{String(event.actor_email ?? "system")}</div>
              <div>
                <div className="flex items-center gap-2 text-sm">
                  <Database className="h-4 w-4 text-brand" />
                  {String(event.action)} · {String(event.resource)}
                </div>
                <pre className="mt-2 max-h-28 overflow-auto rounded-md bg-muted p-2 text-xs text-foreground/65">
                  {JSON.stringify({ old: event.old_values, new: event.new_values, metadata: event.metadata }, null, 2)}
                </pre>
              </div>
              <div className="text-sm text-foreground/55">{formatDate(event.created_at)}</div>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
