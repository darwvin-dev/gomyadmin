import Link from "next/link"
import { KeyRound, Settings, Shield, Users } from "lucide-react"

const items = [
  { href: "/admin/settings/profile", label: "Profile", description: "Name, email, password, and 2FA settings", icon: Settings },
  { href: "/admin/settings/team", label: "Team", description: "Invites, tenant membership, and session controls", icon: Users },
  { href: "/admin/settings/roles", label: "Roles", description: "RBAC bundles and action-level permissions", icon: Shield },
  { href: "/admin/settings/api", label: "API", description: "OpenAPI, tokens, CORS, and webhook readiness", icon: KeyRound }
]

export default function SettingsPage() {
  return (
    <div className="grid gap-5">
      <div>
        <h1 className="text-2xl font-semibold">Settings</h1>
        <p className="mt-1 text-sm text-foreground/58">Security, team, and API controls for a production admin workspace.</p>
      </div>
      <section className="grid gap-3 md:grid-cols-2">
        {items.map((item) => {
          const Icon = item.icon
          return (
            <Link key={item.href} href={item.href} className="rounded-lg border border-border bg-panel p-4 shadow-panel hover:bg-muted/40">
              <Icon className="h-5 w-5 text-brand" />
              <h2 className="mt-3 font-semibold">{item.label}</h2>
              <p className="mt-1 text-sm text-foreground/55">{item.description}</p>
            </Link>
          )
        })}
      </section>
    </div>
  )
}
