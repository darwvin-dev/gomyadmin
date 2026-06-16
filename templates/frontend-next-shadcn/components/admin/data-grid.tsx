"use client"

import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
  type SortingState
} from "@tanstack/react-table"
import { Download, MoreHorizontal, RefreshCw, Trash2, Upload, X } from "lucide-react"
import Link from "next/link"
import { useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import type { FieldMeta, RecordRow, ResourceMeta } from "@/lib/api"
import { formatDate, formatMoney } from "@/lib/utils"
import { RelationLabel } from "@/components/admin/relation-field"

export function DataGrid({
  resource,
  rows,
  loading,
  total,
  page,
  perPage,
  onPage,
  onRefresh,
  onSearch,
  onSort,
  onFilter,
  onBulkDelete
}: {
  resource: ResourceMeta
  rows: RecordRow[]
  loading: boolean
  total: number
  page: number
  perPage: number
  onPage: (page: number) => void
  onRefresh: () => void
  onSearch: (value: string) => void
  onSort: (value: string) => void
  onFilter: (field: string, operator: string, value: string) => void
  onBulkDelete: (ids: string[]) => void
}) {
  const [sorting, setSorting] = useState<SortingState>([])
  const [selected, setSelected] = useState<Record<string, boolean>>({})
  const [filterField, setFilterField] = useState("")
  const [filterOperator, setFilterOperator] = useState("eq")
  const [filterValue, setFilterValue] = useState("")
  const visibleFields = resource.fields.filter((field) => !field.hidden)
  const filterableFields = visibleFields.filter((field) => field.filterable)
  const selectedIDs = rows.map((row) => String(row.id ?? "")).filter((id) => selected[id])
  const allVisibleSelected = rows.length > 0 && rows.every((row) => selected[String(row.id ?? "")])
  const columns = useMemo<ColumnDef<RecordRow>[]>(() => {
    return [
      {
        id: "select",
        header: () => (
          <input
            aria-label="Select all rows"
            checked={allVisibleSelected}
            type="checkbox"
            onChange={(event) => {
              const checked = event.target.checked
              setSelected((value) => {
                const next = { ...value }
                rows.forEach((row) => {
                  const id = String(row.id ?? "")
                  if (id) next[id] = checked
                })
                return next
              })
            }}
          />
        ),
        cell: ({ row }) => {
          const id = String(row.original.id ?? "")
          return (
            <input
              aria-label="Select row"
              checked={Boolean(selected[id])}
              type="checkbox"
              onChange={(event) => setSelected((value) => ({ ...value, [id]: event.target.checked }))}
            />
          )
        },
        size: 36
      },
      ...visibleFields.map((field): ColumnDef<RecordRow> => ({
        accessorKey: field.name,
        header: field.label,
        cell: ({ row }) => <Cell field={field} value={row.original[field.name]} />
      })),
      {
        id: "actions",
        header: "",
        cell: ({ row }) => (
          <Link href={`/admin/resources/${resource.name}/${String(row.original.id)}`} className="inline-flex h-8 w-8 items-center justify-center rounded-md hover:bg-muted">
            <MoreHorizontal className="h-4 w-4" />
          </Link>
        )
      }
    ]
  }, [allVisibleSelected, resource.name, rows, selected, visibleFields])

  const table = useReactTable({
    data: rows,
    columns,
    state: { sorting },
    manualSorting: true,
    manualPagination: true,
    getCoreRowModel: getCoreRowModel(),
    onSortingChange: (updater) => {
      const next = typeof updater === "function" ? updater(sorting) : updater
      setSorting(next)
      const first = next[0]
      if (first) onSort(`${first.desc ? "-" : ""}${first.id}`)
    }
  })

  const pages = Math.max(1, Math.ceil(total / perPage))

  return (
    <section className="overflow-hidden rounded-lg border border-border bg-panel shadow-panel">
      <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
        <Input className="max-w-sm" placeholder={`Search ${resource.label.toLowerCase()}`} onChange={(event) => onSearch(event.target.value)} />
        {filterableFields.length > 0 && (
          <div className="flex flex-wrap items-center gap-2">
            <select className="h-8 rounded-md border border-border bg-panel px-2 text-sm" value={filterField} onChange={(event) => setFilterField(event.target.value)}>
              <option value="">Filter field</option>
              {filterableFields.map((field) => (
                <option key={field.name} value={field.name}>
                  {field.label}
                </option>
              ))}
            </select>
            <select className="h-8 rounded-md border border-border bg-panel px-2 text-sm" value={filterOperator} onChange={(event) => setFilterOperator(event.target.value)}>
              <option value="eq">equals</option>
              <option value="contains">contains</option>
              <option value="starts_with">starts with</option>
              <option value="ends_with">ends with</option>
              <option value="gte">at least</option>
              <option value="lte">at most</option>
            </select>
            <Input className="h-8 w-40" placeholder="Value" value={filterValue} onChange={(event) => setFilterValue(event.target.value)} />
            <Button variant="outline" size="sm" onClick={() => filterField && onFilter(filterField, filterOperator, filterValue)}>
              Apply
            </Button>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Clear filter"
              onClick={() => {
                setFilterField("")
                setFilterValue("")
                onFilter("", "eq", "")
              }}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        )}
        <Button variant="outline" size="sm">
          <Upload className="h-4 w-4" />
          Import
        </Button>
        <Button variant="outline" size="sm" asChild>
          <a href={`${process.env.NEXT_PUBLIC_ADMIN_API_URL ?? "http://localhost:8080"}/admin/api/${resource.name}/export`}>
            <Download className="h-4 w-4" />
            Export
          </a>
        </Button>
        <Button className="ml-auto" variant="outline" size="icon" onClick={onRefresh} aria-label="Refresh">
          <RefreshCw className="h-4 w-4" />
        </Button>
        <Button
          variant="danger"
          size="sm"
          disabled={selectedIDs.length === 0}
          onClick={() => {
            onBulkDelete(selectedIDs)
            setSelected({})
          }}
        >
          <Trash2 className="h-4 w-4" />
          Delete {selectedIDs.length > 0 ? selectedIDs.length : ""}
        </Button>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[760px] border-collapse text-sm">
          <thead className="bg-muted/55 text-left text-xs uppercase tracking-normal text-foreground/55">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th key={header.id} className="h-10 border-b border-border px-3 font-medium">
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {loading &&
              Array.from({ length: 6 }).map((_, index) => (
                <tr key={index}>
                  {table.getAllColumns().map((column) => (
                    <td key={column.id} className="border-b border-border px-3 py-3">
                      <div className="h-4 w-full max-w-32 animate-pulse rounded bg-muted" />
                    </td>
                  ))}
                </tr>
              ))}
            {!loading &&
              table.getRowModel().rows.map((row) => (
                <tr key={row.id} className="hover:bg-muted/45">
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="border-b border-border px-3 py-2.5">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={columns.length} className="px-3 py-16 text-center text-sm text-foreground/55">
                  No records match this view.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <div className="flex items-center justify-between px-3 py-3 text-sm text-foreground/60">
        <span>
          Page {page} of {pages} · {total} records
        </span>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPage(page - 1)}>
            Previous
          </Button>
          <Button variant="outline" size="sm" disabled={page >= pages} onClick={() => onPage(page + 1)}>
            Next
          </Button>
        </div>
      </div>
    </section>
  )
}

function Cell({ field, value }: { field: FieldMeta; value: unknown }) {
  if (field.type === "relation" && field.relation) return <RelationLabel relation={field.relation} value={value} />
  if (field.type === "status" || field.type === "enum") return <Badge value={value} />
  if (field.type === "datetime" || field.type === "date") return <span>{formatDate(value)}</span>
  if (field.type === "money") return <span className="font-medium">{formatMoney(value)}</span>
  return <span className="text-foreground/82">{String(value ?? "")}</span>
}
