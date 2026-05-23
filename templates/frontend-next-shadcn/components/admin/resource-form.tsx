"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Save } from "lucide-react"
import { useRouter } from "next/navigation"
import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api, type RecordRow, type ResourceMeta } from "@/lib/api"

export function ResourceForm({ resource, id }: { resource: string; id?: string }) {
  const router = useRouter()
  const queryClient = useQueryClient()
  const resources = useQuery({ queryKey: ["resources"], queryFn: async () => (await api.resources()).data ?? [] })
  const meta = resources.data?.find((item) => item.name === resource)
  const record = useQuery({
    queryKey: ["record", resource, id],
    enabled: Boolean(id),
    queryFn: async () => (await api.get(resource, id ?? "")).data ?? {}
  })
  const [draft, setDraft] = useState<RecordRow>({})
  const mutation = useMutation({
    mutationFn: async () => {
      const payload = { ...(record.data ?? {}), ...draft }
      if (id) return (await api.update(resource, id, payload)).data
      return (await api.create(resource, payload)).data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["resource", resource] })
      router.push(`/admin/resources/${resource}`)
    }
  })

  if (!meta) {
    return <div className="rounded-lg border border-border bg-panel p-8 text-sm text-foreground/55">Loading resource definition...</div>
  }

  return (
    <form
      className="grid max-w-3xl gap-5"
      onSubmit={(event) => {
        event.preventDefault()
        mutation.mutate()
      }}
    >
      <div>
        <h1 className="text-2xl font-semibold">{id ? `Edit ${meta.label}` : `New ${meta.label}`}</h1>
        <p className="mt-1 text-sm text-foreground/58">{meta.description}</p>
      </div>
      <section className="rounded-lg border border-border bg-panel p-4 shadow-panel">
        <div className="grid gap-4 md:grid-cols-2">
          {editableFields(meta).map((field) => (
            <label key={field.name} className="grid gap-1.5 text-sm">
              <span className="font-medium">{field.label}</span>
              {field.enum_values?.length ? (
                <select
                  className="h-9 rounded-md border border-border bg-panel px-3 text-sm"
                  defaultValue={String((record.data ?? {})[field.name] ?? "")}
                  onChange={(event) => setDraft((value) => ({ ...value, [field.name]: event.target.value }))}
                >
                  <option value="">Select</option>
                  {field.enum_values.map((value) => (
                    <option key={value} value={value}>
                      {value.replaceAll("_", " ")}
                    </option>
                  ))}
                </select>
              ) : (
                <Input
                  defaultValue={String((record.data ?? {})[field.name] ?? "")}
                  onChange={(event) => setDraft((value) => ({ ...value, [field.name]: event.target.value }))}
                />
              )}
            </label>
          ))}
        </div>
      </section>
      <div className="flex gap-2">
        <Button disabled={mutation.isPending}>
          <Save className="h-4 w-4" />
          Save
        </Button>
        <Button type="button" variant="outline" onClick={() => router.back()}>
          Cancel
        </Button>
      </div>
    </form>
  )
}

function editableFields(resource: ResourceMeta) {
  return resource.fields.filter((field) => !field.hidden && !field.readonly && field.name !== "id" && field.name !== "created_at")
}
