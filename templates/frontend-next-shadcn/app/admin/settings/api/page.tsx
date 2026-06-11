"use client"

import { KeyRound, RotateCcw } from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api, type APIKeyRow } from "@/lib/api"

export default function APISettingsPage() {
  const [keys, setKeys] = useState<APIKeyRow[]>([])
  const [name, setName] = useState("Automation")
  const [scopes, setScopes] = useState("*.view, *.create, *.update")
  const [expiresIn, setExpiresIn] = useState("720h")
  const [revealedSecret, setRevealedSecret] = useState("")
  const [isSaving, setIsSaving] = useState(false)

  const hasKeys = keys.length > 0
  const parsedScopes = useMemo(
    () => scopes.split(",").map((value) => value.trim()).filter(Boolean),
    [scopes]
  )

  async function loadKeys() {
    const response = await api.apiKeys()
    setKeys(response.data ?? [])
  }

  useEffect(() => {
    loadKeys().catch(() => setKeys([]))
  }, [])

  return (
    <div className="grid max-w-5xl gap-6">
      <div>
        <h1 className="text-2xl font-semibold">API</h1>
        <p className="mt-1 text-sm text-foreground/58">OAuth entry points, machine access, and one-time API secrets.</p>
      </div>

      <section className="grid gap-4 rounded-lg border border-border bg-panel p-4 shadow-panel">
        <div className="flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-md bg-brand text-white">
            <KeyRound className="h-5 w-5" />
          </div>
          <div>
            <h2 className="font-semibold">Create API key</h2>
            <p className="text-sm text-foreground/55">Secrets are shown once. Store them in your runtime, not in source control.</p>
          </div>
        </div>
        <div className="grid gap-3 md:grid-cols-3">
          <Input aria-label="Key name" value={name} onChange={(event) => setName(event.target.value)} />
          <Input aria-label="Scopes" value={scopes} onChange={(event) => setScopes(event.target.value)} />
          <Input aria-label="Expires in" value={expiresIn} onChange={(event) => setExpiresIn(event.target.value)} />
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Button
            disabled={isSaving}
            onClick={async () => {
              setIsSaving(true)
              try {
                const response = await api.createAPIKey({ name, scopes: parsedScopes, expires_in: expiresIn })
                setRevealedSecret(response.data?.secret ?? "")
                await loadKeys()
              } finally {
                setIsSaving(false)
              }
            }}
          >
            Generate key
          </Button>
          {revealedSecret ? (
            <code className="rounded-md border border-border bg-background px-3 py-2 text-xs">{revealedSecret}</code>
          ) : null}
        </div>
      </section>

      <section className="grid gap-3 rounded-lg border border-border bg-panel p-4 shadow-panel">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="font-semibold">Issued keys</h2>
            <p className="text-sm text-foreground/55">Rotate or revoke keys without affecting browser sessions.</p>
          </div>
          <Button variant="outline" size="icon" onClick={() => loadKeys()}>
            <RotateCcw className="h-4 w-4" />
          </Button>
        </div>
        {hasKeys ? (
          <div className="grid gap-2">
            {keys.map((key) => (
              <div key={key.id} className="grid gap-3 rounded-md border border-border bg-background p-3 md:grid-cols-[1.4fr_1fr_auto] md:items-center">
                <div>
                  <div className="font-medium">{key.name}</div>
                  <div className="text-xs text-foreground/55">
                    {key.prefix} • {key.scopes.join(", ") || "full actor permissions"}
                  </div>
                </div>
                <div className="text-xs text-foreground/55">
                  <div>Created: {new Date(key.created_at).toLocaleString()}</div>
                  <div>Last used: {key.last_used_at ? new Date(key.last_used_at).toLocaleString() : "Never"}</div>
                </div>
                <Button
                  variant="outline"
                  onClick={async () => {
                    await api.revokeAPIKey(key.id)
                    await loadKeys()
                  }}
                >
                  Revoke
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <div className="rounded-md border border-dashed border-border px-4 py-8 text-sm text-foreground/55">
            No API keys issued yet.
          </div>
        )}
      </section>
    </div>
  )
}
