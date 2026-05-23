"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { Building2, Command, CreditCard, Database, FileText, Gauge, LifeBuoy, List, Moon, Search, Settings, Sun, Users } from "lucide-react"
import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

const nav = [
  { href: "/admin/dashboard", label: "Dashboard", icon: Gauge },
  { href: "/admin/resources/users", label: "Users", icon: Users },
  { href: "/admin/resources/customers", label: "Customers", icon: Building2 },
  { href: "/admin/resources/invoices", label: "Invoices", icon: FileText },
  { href: "/admin/resources/invoice_items", label: "Invoice Items", icon: List },
  { href: "/admin/resources/payments", label: "Payments", icon: CreditCard },
  { href: "/admin/resources/tickets", label: "Tickets", icon: LifeBuoy },
  { href: "/admin/audit", label: "Audit Log", icon: Database },
  { href: "/admin/files", label: "Files", icon: FileText },
  { href: "/admin/settings", label: "Settings", icon: Settings }
]

export function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const [dark, setDark] = useState(false)

  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark)
  }, [dark])

  if (pathname === "/admin/login") {
    return children
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r border-border bg-panel/92 px-3 py-4 backdrop-blur lg:block">
        <Link href="/admin/dashboard" className="mb-5 flex h-10 items-center gap-2 rounded-md px-2">
          <div className="grid h-7 w-7 place-items-center rounded-md bg-brand text-sm font-bold text-white">G</div>
          <div>
            <div className="text-sm font-semibold">GoMyAdmin</div>
            <div className="text-xs text-foreground/55">CRM admin</div>
          </div>
        </Link>
        <nav className="grid gap-1">
          {nav.map((item) => {
            const active = pathname.startsWith(item.href)
            const Icon = item.icon
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex h-9 items-center gap-2 rounded-md px-2 text-sm text-foreground/70 transition hover:bg-muted hover:text-foreground",
                  active && "bg-muted text-foreground"
                )}
              >
                <Icon className="h-4 w-4" />
                {item.label}
              </Link>
            )
          })}
        </nav>
      </aside>
      <div className="lg:pl-64">
        <header className="sticky top-0 z-40 flex h-14 items-center gap-3 border-b border-border bg-background/88 px-4 backdrop-blur">
          <div className="hidden min-w-0 items-center gap-2 text-sm text-foreground/60 md:flex">
            <Command className="h-4 w-4" />
            <span>Admin</span>
            <span>/</span>
            <span className="truncate text-foreground">{pathname.split("/").filter(Boolean).slice(-1)[0] ?? "dashboard"}</span>
          </div>
          <div className="ml-auto flex items-center gap-2">
            <div className="relative hidden w-72 md:block">
              <Search className="pointer-events-none absolute left-2.5 top-2.5 h-4 w-4 text-foreground/40" />
              <Input className="pl-8" placeholder="Search resources, actions, records" />
            </div>
            <select className="h-9 rounded-md border border-border bg-panel px-2 text-sm">
              <option>Northstar CRM</option>
              <option>Acme Telecom</option>
            </select>
            <Button variant="outline" size="icon" onClick={() => setDark((value) => !value)} aria-label="Toggle theme">
              {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </Button>
            <Button variant="outline">Darwin</Button>
          </div>
        </header>
        <main className="mx-auto w-full max-w-[1500px] px-4 py-5">{children}</main>
      </div>
    </div>
  )
}
