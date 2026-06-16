"use client"

import { useQuery } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { api, type RecordRow, type RelationMeta } from "@/lib/api"

// Fields tried, in order, when a relation has no explicit display_field.
const FALLBACK_LABEL_FIELDS = ["name", "title", "label", "email"]

/**
 * Pick a human-readable label for a related record. Uses the relation's
 * display_field when present, then a few common name-like fields, and finally
 * falls back to the record id so the UI is never blank.
 */
export function relationLabel(record: RecordRow | undefined, relation: RelationMeta, id: string): string {
  if (!record) return id
  const candidates = relation.display_field ? [relation.display_field, ...FALLBACK_LABEL_FIELDS] : FALLBACK_LABEL_FIELDS
  for (const key of candidates) {
    const value = record[key]
    if (typeof value === "string" && value.trim()) return value
    if (typeof value === "number") return String(value)
  }
  return id
}

type RelationOption = { id: string; label: string }

/**
 * Fetch the target resource once (deduped by React Query) and expose an
 * id -> label map plus ready-to-render options for a relation field.
 */
export function useRelationOptions(relation: RelationMeta) {
  const query = useQuery({
    queryKey: ["relation-options", relation.resource],
    queryFn: async () => (await api.list(relation.resource, new URLSearchParams())).data ?? []
  })
  const records = query.data ?? []
  const labelFor = (id: string) => {
    const match = records.find((row) => String(row.id ?? "") === id)
    return relationLabel(match, relation, id)
  }
  const options: RelationOption[] = records.map((row) => {
    const id = String(row.id ?? "")
    return { id, label: relationLabel(row, relation, id) }
  })
  return { options, labelFor, isLoading: query.isLoading }
}

/** Read-only label for a belongs-to value in list/detail views. */
export function RelationLabel({ relation, value }: { relation: RelationMeta; value: unknown }) {
  const id = value == null ? "" : String(value)
  const { labelFor } = useRelationOptions(relation)
  if (!id) return <span className="text-foreground/40">—</span>
  return <span className="text-foreground/82">{labelFor(id)}</span>
}

/** Native select for choosing a belongs-to value in forms. */
export function RelationSelect({
  relation,
  value,
  onChange
}: {
  relation: RelationMeta
  value: unknown
  onChange: (value: unknown) => void
}) {
  const { options, labelFor } = useRelationOptions(relation)
  const [selected, setSelected] = useState(value == null ? "" : String(value))
  // The record value loads asynchronously, so the form mounts this select
  // before `value` is known; sync state when it arrives (and on later resets).
  useEffect(() => {
    setSelected(value == null ? "" : String(value))
  }, [value])
  // Controlled select: the options list also loads asynchronously, so a plain
  // defaultValue would be lost before the matching option exists. Keep the
  // current value's option present even while the target list is loading.
  const hasSelected = selected === "" || options.some((option) => option.id === selected)
  return (
    <select
      className="h-9 rounded-md border border-border bg-panel px-3 text-sm"
      value={selected}
      onChange={(event) => {
        setSelected(event.target.value)
        onChange(event.target.value)
      }}
    >
      <option value="">Select</option>
      {!hasSelected && <option value={selected}>{labelFor(selected)}</option>}
      {options.map((option) => (
        <option key={option.id} value={option.id}>
          {option.label}
        </option>
      ))}
    </select>
  )
}
