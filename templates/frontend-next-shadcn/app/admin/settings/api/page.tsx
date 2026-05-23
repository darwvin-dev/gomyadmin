import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export default function APISettingsPage() {
  return (
    <div className="grid max-w-3xl gap-5">
      <div>
        <h1 className="text-2xl font-semibold">API</h1>
        <p className="mt-1 text-sm text-foreground/58">OpenAPI, CORS, webhook, and token controls.</p>
      </div>
      <section className="grid gap-3 rounded-lg border border-border bg-panel p-4 shadow-panel">
        <Input defaultValue="http://localhost:3000" />
        <Input defaultValue="/admin/api/openapi.json" />
      </section>
      <Button className="w-fit">Regenerate API token</Button>
    </div>
  )
}
