import { Shield } from "lucide-react"

const roles = ["super_admin", "tenant_admin", "manager", "support", "viewer"]

export default function RoleSettingsPage() {
  return (
    <div className="grid gap-5">
      <div>
        <h1 className="text-2xl font-semibold">Roles</h1>
        <p className="mt-1 text-sm text-foreground/58">Backend-enforced RBAC and row-level policies.</p>
      </div>
      <section className="grid gap-3 md:grid-cols-2">
        {roles.map((role) => (
          <div key={role} className="rounded-lg border border-border bg-panel p-4 shadow-panel">
            <Shield className="h-5 w-5 text-brand" />
            <h2 className="mt-3 font-semibold">{role}</h2>
            <p className="mt-1 text-sm text-foreground/55">Permissions are checked on every backend mutation and action.</p>
          </div>
        ))}
      </section>
    </div>
  )
}
