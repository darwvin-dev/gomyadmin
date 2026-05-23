"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { ArrowRight, LockKeyhole } from "lucide-react"
import { useRouter } from "next/navigation"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api"

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(1)
})

type LoginInput = z.infer<typeof schema>

export default function LoginPage() {
  const router = useRouter()
  const form = useForm<LoginInput>({
    resolver: zodResolver(schema),
    defaultValues: { email: "admin@example.com", password: "password" }
  })

  return (
    <main className="grid min-h-screen place-items-center bg-background px-4">
      <form
        className="w-full max-w-sm rounded-lg border border-border bg-panel p-6 shadow-panel"
        onSubmit={form.handleSubmit(async (values) => {
          await api.login(values.email, values.password)
          router.push("/admin/dashboard")
        })}
      >
        <div className="mb-5 flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-md bg-brand text-white">
            <LockKeyhole className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-lg font-semibold">GoMyAdmin</h1>
            <p className="text-sm text-foreground/55">Secure admin workspace</p>
          </div>
        </div>
        <div className="grid gap-3">
          <Input aria-label="Email" {...form.register("email")} />
          <Input aria-label="Password" type="password" {...form.register("password")} />
          <Button disabled={form.formState.isSubmitting}>
            Continue
            <ArrowRight className="h-4 w-4" />
          </Button>
        </div>
      </form>
    </main>
  )
}
