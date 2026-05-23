import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export default function TeamSettingsPage() {
  return (
    <div className="grid max-w-3xl gap-5">
      <div>
        <h1 className="text-2xl font-semibold">Team</h1>
        <p className="mt-1 text-sm text-foreground/58">Invite teammates and control tenant membership.</p>
      </div>
      <section className="grid gap-3 rounded-lg border border-border bg-panel p-4 shadow-panel">
        <Input placeholder="teammate@example.com" />
        <select className="h-9 rounded-md border border-border bg-panel px-3 text-sm">
          <option>support</option>
          <option>manager</option>
          <option>tenant_admin</option>
        </select>
      </section>
      <Button className="w-fit">Send invite</Button>
    </div>
  )
}
