"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Plus } from "lucide-react"
import Link from "next/link"
import { useParams } from "next/navigation"
import { useMemo, useState, useTransition } from "react"
import { DataGrid } from "@/components/admin/data-grid"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api"

export default function ResourcePage() {
  const params = useParams<{ resource: string }>()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState("")
  const [sort, setSort] = useState("-created_at")
  const [filter, setFilter] = useState<{ field: string; operator: string; value: string }>({ field: "", operator: "eq", value: "" })
  const [, startTransition] = useTransition()
  const queryClient = useQueryClient()
  const resources = useQuery({ queryKey: ["resources"], queryFn: async () => (await api.resources()).data ?? [] })
  const meta = resources.data?.find((resource) => resource.name === params.resource)
  const query = useMemo(() => {
    const values = new URLSearchParams()
    values.set("page", String(page))
    values.set("per_page", "25")
    values.set("sort", sort)
    if (search) values.set("q", search)
    if (filter.field && filter.value) values.set(`filter[${filter.field}][${filter.operator}]`, filter.value)
    return values
  }, [filter, page, search, sort])
  const records = useQuery({
    queryKey: ["resource", params.resource, query.toString()],
    enabled: Boolean(meta),
    queryFn: async () => api.list(params.resource, query)
  })
  const bulkDelete = useMutation({
    mutationFn: async (ids: string[]) => api.bulkDelete(params.resource, ids),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["resource", params.resource] })
      await queryClient.invalidateQueries({ queryKey: ["audit"] })
    }
  })

  if (!meta) {
    return <div className="rounded-lg border border-border bg-panel p-8 text-sm text-foreground/55">Loading resource...</div>
  }

  return (
    <div className="grid gap-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">{meta.label}</h1>
          <p className="mt-1 text-sm text-foreground/58">{meta.description}</p>
        </div>
        <Button asChild>
          <Link href={`/admin/resources/${params.resource}/new`}>
            <Plus className="h-4 w-4" />
            New {meta.label}
          </Link>
        </Button>
      </div>
      <div className="flex gap-2 overflow-x-auto">
        {["All records", "Active customers", "Blocked users", "Invoices this month", "Failed payments"].map((view) => (
          <button key={view} className="h-8 shrink-0 rounded-md border border-border bg-panel px-3 text-sm text-foreground/70 hover:bg-muted">
            {view}
          </button>
        ))}
      </div>
      <DataGrid
        resource={meta}
        rows={records.data?.data ?? []}
        loading={records.isLoading}
        total={Number(records.data?.meta.total ?? 0)}
        page={page}
        perPage={25}
        onPage={setPage}
        onRefresh={() => records.refetch()}
        onSearch={(value) => startTransition(() => setSearch(value))}
        onSort={setSort}
        onFilter={(field, operator, value) => {
          setPage(1)
          setFilter({ field, operator, value })
        }}
        onBulkDelete={(ids) => bulkDelete.mutate(ids)}
      />
    </div>
  )
}
