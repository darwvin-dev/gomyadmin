"use client"

import { useParams } from "next/navigation"
import { ResourceForm } from "@/components/admin/resource-form"

export default function NewResourcePage() {
  const params = useParams<{ resource: string }>()
  return <ResourceForm resource={params.resource} />
}
