import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export default function ProfileSettingsPage() {
  return (
    <SettingsForm title="Profile" description="Update administrator identity and account security.">
      <Input defaultValue="Darwin Admin" />
      <Input defaultValue="admin@example.com" />
      <Input placeholder="New password" type="password" />
    </SettingsForm>
  )
}

function SettingsForm({ title, description, children }: { title: string; description: string; children: React.ReactNode }) {
  return (
    <div className="grid max-w-2xl gap-5">
      <div>
        <h1 className="text-2xl font-semibold">{title}</h1>
        <p className="mt-1 text-sm text-foreground/58">{description}</p>
      </div>
      <section className="grid gap-3 rounded-lg border border-border bg-panel p-4 shadow-panel">{children}</section>
      <Button className="w-fit">Save changes</Button>
    </div>
  )
}
