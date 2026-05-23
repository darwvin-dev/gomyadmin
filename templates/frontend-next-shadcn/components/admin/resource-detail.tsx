"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Ban, KeyRound, Pencil, Trash2, Undo2 } from "lucide-react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api"
import { formatDate, formatMoney } from "@/lib/utils"

export function ResourceDetail({ resource, id }: { resource: string; id: string }) {
  const router = useRouter()
  const queryClient = useQueryClient()
  const resources = useQuery({ queryKey: ["resources"], queryFn: async () => (await api.resources()).data ?? [] })
  const meta = resources.data?.find((item) => item.name === resource)
  const record = useQuery({ queryKey: ["record", resource, id], queryFn: async () => (await api.get(resource, id)).data ?? {} })
  const action = useMutation({
    mutationFn: async (name: string) => (await api.action(resource, id, name, { reason: "Requested from admin UI" })).data,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["record", resource, id] })
      await queryClient.invalidateQueries({ queryKey: ["audit"] })
    }
  })
  const remove = useMutation({
    mutationFn: async () => api.remove(resource, id),
    onSuccess: () => router.push(`/admin/resources/${resource}`)
  })

  if (!meta || !record.data) {
    return <div className="rounded-lg border border-border bg-panel p-8 text-sm text-foreground/55">Loading record...</div>
  }

  return (
    <div className="grid gap-5 xl:grid-cols-[1fr_360px]">
      <section className="grid gap-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold">{meta.label} detail</h1>
            <p className="mt-1 text-sm text-foreground/58">{String(record.data.id)}</p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" asChild>
              <Link href={`/admin/resources/${resource}/${id}/edit`}>
                <Pencil className="h-4 w-4" />
                Edit
              </Link>
            </Button>
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="danger">
                  <Trash2 className="h-4 w-4" />
                  Delete
                </Button>
              </DialogTrigger>
              <DialogContent title="Delete record">
                <div className="grid gap-4">
                  <p className="text-sm text-foreground/65">This mutation is audited and cannot be undone from the UI.</p>
                  <Button variant="danger" onClick={() => remove.mutate()}>Confirm delete</Button>
                </div>
              </DialogContent>
            </Dialog>
          </div>
        </div>
        <div className="rounded-lg border border-border bg-panel shadow-panel">
          <div className="grid divide-y divide-border">
            {meta.fields.filter((field) => !field.hidden).map((field) => (
              <div key={field.name} className="grid gap-2 px-4 py-3 md:grid-cols-[220px_1fr]">
                <div className="text-sm text-foreground/55">{field.label}</div>
                <div className="text-sm font-medium">{renderValue(field.type, record.data[field.name])}</div>
              </div>
            ))}
          </div>
        </div>
      </section>
      <aside className="grid content-start gap-4">
        <div className="rounded-lg border border-border bg-panel p-4 shadow-panel">
          <h2 className="font-semibold">Actions</h2>
          <div className="mt-3 grid gap-2">
            {meta.actions.map((item) => (
              <Button key={item.name} variant={item.danger ? "danger" : "outline"} onClick={() => action.mutate(item.name)}>
                {item.name.includes("password") ? <KeyRound className="h-4 w-4" /> : item.name.includes("refund") ? <Undo2 className="h-4 w-4" /> : <Ban className="h-4 w-4" />}
                {item.label}
              </Button>
            ))}
          </div>
        </div>
        <div className="rounded-lg border border-border bg-panel p-4 shadow-panel">
          <h2 className="font-semibold">Action reason</h2>
          <Input className="mt-3" placeholder="Required for danger actions" />
        </div>
      </aside>
    </div>
  )
}

function renderValue(type: string, value: unknown) {
  if (type === "status" || type === "enum") return <Badge value={value} />
  if (type === "datetime" || type === "date") return formatDate(value)
  if (type === "money") return formatMoney(value)
  return String(value ?? "")
}
