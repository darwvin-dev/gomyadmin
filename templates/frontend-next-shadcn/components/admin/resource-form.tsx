"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Save } from "lucide-react"
import { useRouter } from "next/navigation"
import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api, type FieldMeta, type RecordRow, type ResourceMeta } from "@/lib/api"
import { RelationSelect } from "@/components/admin/relation-field"

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
              <FieldInput field={field} value={(record.data ?? {})[field.name]} onChange={(next) => setDraft((value) => ({ ...value, [field.name]: next }))} />
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

function FieldInput({ field, value, onChange }: { field: FieldMeta; value: unknown; onChange: (value: unknown) => void }) {
  if (field.type === "relation" && field.relation) {
    return <RelationSelect relation={field.relation} value={value} onChange={onChange} />
  }
  if (field.enum_values?.length) {
    return (
      <select className="h-9 rounded-md border border-border bg-panel px-3 text-sm" defaultValue={String(value ?? "")} onChange={(event) => onChange(event.target.value)}>
        <option value="">Select</option>
        {field.enum_values.map((item) => (
          <option key={item} value={item}>
            {item.replaceAll("_", " ")}
          </option>
        ))}
      </select>
    )
  }
  if (field.type === "boolean") {
    return (
      <label className="flex h-9 items-center gap-2 rounded-md border border-border bg-panel px-3">
        <input defaultChecked={Boolean(value)} type="checkbox" onChange={(event) => onChange(event.target.checked)} />
        <span className="text-sm text-foreground/65">Enabled</span>
      </label>
    )
  }
  if (field.type === "text" || field.type === "markdown" || field.type === "rich_text" || field.type === "json" || field.type === "jsonb") {
    return (
      <textarea
        className="min-h-28 rounded-md border border-border bg-panel px-3 py-2 text-sm outline-none transition placeholder:text-foreground/40 focus:border-brand focus:ring-2 focus:ring-brand/20"
        defaultValue={String(value ?? "")}
        onChange={(event) => onChange(event.target.value)}
      />
    )
  }
  return (
    <Input
      defaultValue={String(value ?? "")}
      type={inputType(field.type)}
      onChange={(event) => onChange(field.type === "integer" || field.type === "number" || field.type === "float" || field.type === "decimal" ? Number(event.target.value) : event.target.value)}
    />
  )
}

function inputType(type: string) {
  switch (type) {
    case "email":
      return "email"
    case "password":
      return "password"
    case "date":
      return "date"
    case "datetime":
      return "datetime-local"
    case "integer":
    case "number":
    case "float":
    case "decimal":
    case "money":
      return "number"
    default:
      return "text"
  }
}

function editableFields(resource: ResourceMeta) {
  return resource.fields.filter((field) => !field.hidden && !field.readonly && field.name !== "id" && field.name !== "created_at")
}
