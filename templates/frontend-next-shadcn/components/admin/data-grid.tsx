"use client"

import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
  type SortingState
} from "@tanstack/react-table"
import { Download, MoreHorizontal, RefreshCw, SlidersHorizontal, Upload } from "lucide-react"
import Link from "next/link"
import { useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import type { FieldMeta, RecordRow, ResourceMeta } from "@/lib/api"
import { formatDate, formatMoney } from "@/lib/utils"

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
  onSort
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
}) {
  const [sorting, setSorting] = useState<SortingState>([])
  const visibleFields = resource.fields.filter((field) => !field.hidden)
  const columns = useMemo<ColumnDef<RecordRow>[]>(() => {
    return [
      {
        id: "select",
        header: () => <input aria-label="Select all rows" type="checkbox" />,
        cell: () => <input aria-label="Select row" type="checkbox" />,
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
  }, [resource.name, visibleFields])

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
        <Button variant="outline" size="sm">
          <SlidersHorizontal className="h-4 w-4" />
          View
        </Button>
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
  if (field.type === "status" || field.type === "enum") return <Badge value={value} />
  if (field.type === "datetime" || field.type === "date") return <span>{formatDate(value)}</span>
  if (field.type === "money") return <span className="font-medium">{formatMoney(value)}</span>
  return <span className="text-foreground/82">{String(value ?? "")}</span>
}
