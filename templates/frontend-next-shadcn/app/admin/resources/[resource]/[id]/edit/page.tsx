"use client"

import { useParams } from "next/navigation"
import { ResourceForm } from "@/components/admin/resource-form"

export default function EditResourcePage() {
  const params = useParams<{ resource: string; id: string }>()
  return <ResourceForm resource={params.resource} id={params.id} />
}
