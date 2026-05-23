"use client"

import { useParams } from "next/navigation"
import { ResourceDetail } from "@/components/admin/resource-detail"

export default function ResourceDetailPage() {
  const params = useParams<{ resource: string; id: string }>()
  return <ResourceDetail resource={params.resource} id={params.id} />
}
