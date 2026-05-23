"use client"

import { useQuery } from "@tanstack/react-query"
import Link from "next/link"
import { Activity, ArrowUpRight, CreditCard, Database, Users } from "lucide-react"
import { api } from "@/lib/api"
import { formatDate } from "@/lib/utils"

export default function DashboardPage() {
  const resources = useQuery({ queryKey: ["resources"], queryFn: async () => (await api.resources()).data ?? [] })
  const audit = useQuery({ queryKey: ["audit"], queryFn: async () => (await api.audit()).data ?? [] })

  const stats = [
    { label: "Active resources", value: resources.data?.length ?? 0, icon: Database },
    { label: "Admin users", value: "3", icon: Users },
    { label: "Monthly revenue", value: "$4.8k", icon: CreditCard },
    { label: "Audit events", value: audit.data?.length ?? 0, icon: Activity }
  ]

  return (
    <div className="grid gap-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Dashboard</h1>
          <p className="mt-1 text-sm text-foreground/58">Operations overview across tenants, resources, and sensitive admin activity.</p>
        </div>
        <Link href="/admin/resources/users/new" className="inline-flex h-9 items-center gap-2 rounded-md bg-brand px-3 text-sm font-medium text-white">
          Invite user
          <ArrowUpRight className="h-4 w-4" />
        </Link>
      </div>
      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon
          return (
            <div key={stat.label} className="rounded-lg border border-border bg-panel p-4 shadow-panel">
              <div className="flex items-center justify-between">
                <span className="text-sm text-foreground/58">{stat.label}</span>
                <Icon className="h-4 w-4 text-brand" />
              </div>
              <div className="mt-3 text-2xl font-semibold">{stat.value}</div>
            </div>
          )
        })}
      </section>
      <section className="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
        <div className="rounded-lg border border-border bg-panel shadow-panel">
          <div className="border-b border-border px-4 py-3">
            <h2 className="font-semibold">Resources</h2>
          </div>
          <div className="divide-y divide-border">
            {resources.data?.map((resource) => (
              <Link key={resource.name} href={`/admin/resources/${resource.name}`} className="flex items-center justify-between px-4 py-3 hover:bg-muted/50">
                <div>
                  <div className="font-medium">{resource.label}</div>
                  <div className="text-sm text-foreground/55">{resource.description}</div>
                </div>
                <ArrowUpRight className="h-4 w-4 text-foreground/40" />
              </Link>
            ))}
          </div>
        </div>
        <div className="rounded-lg border border-border bg-panel shadow-panel">
          <div className="border-b border-border px-4 py-3">
            <h2 className="font-semibold">Recent audit</h2>
          </div>
          <div className="divide-y divide-border">
            {audit.data?.slice(0, 6).map((event) => (
              <div key={String(event.id)} className="px-4 py-3">
                <div className="text-sm font-medium">{String(event.actor_email ?? "system")} · {String(event.action)}</div>
                <div className="text-xs text-foreground/52">{String(event.resource)} · {formatDate(event.created_at)}</div>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  )
}
