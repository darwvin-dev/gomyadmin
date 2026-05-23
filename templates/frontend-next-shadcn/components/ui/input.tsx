import * as React from "react"
import { cn } from "@/lib/utils"

export function Input({ className, ...props }: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        "h-9 w-full rounded-md border border-border bg-panel px-3 text-sm outline-none transition placeholder:text-foreground/40 focus:border-brand focus:ring-2 focus:ring-brand/20",
        className
      )}
      {...props}
    />
  )
}
