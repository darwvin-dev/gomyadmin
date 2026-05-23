"use client"

import { useQuery } from "@tanstack/react-query"
import { FileUp, Upload } from "lucide-react"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api"
import { formatDate } from "@/lib/utils"

export default function FilesPage() {
  const files = useQuery({ queryKey: ["files"], queryFn: async () => (await api.files()).data ?? [] })

  return (
    <div className="grid gap-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Files</h1>
          <p className="mt-1 text-sm text-foreground/58">Local development storage with S3 and Cloudflare R2-ready backend interfaces.</p>
        </div>
        <Button>
          <Upload className="h-4 w-4" />
          Upload
        </Button>
      </div>
      <section className="rounded-lg border border-dashed border-border bg-panel p-8 text-center shadow-panel">
        <FileUp className="mx-auto h-8 w-8 text-brand" />
        <h2 className="mt-3 font-semibold">Drop files to upload</h2>
        <p className="mx-auto mt-1 max-w-md text-sm text-foreground/55">Images, PDFs, CSV, and text files are validated by MIME type and size before storage.</p>
      </section>
      <section className="rounded-lg border border-border bg-panel shadow-panel">
        <div className="border-b border-border px-4 py-3 text-sm font-medium">Recent uploads</div>
        <div className="divide-y divide-border">
          {files.data?.length === 0 && <div className="px-4 py-8 text-sm text-foreground/55">No files uploaded yet.</div>}
          {files.data?.map((file) => (
            <div key={String(file.id)} className="grid gap-2 px-4 py-3 md:grid-cols-[1fr_160px_180px]">
              <div className="font-medium">{String(file.name)}</div>
              <div className="text-sm text-foreground/55">{String(file.content_type)}</div>
              <div className="text-sm text-foreground/55">{formatDate(file.created_at)}</div>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
