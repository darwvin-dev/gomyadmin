import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cn } from "@/lib/utils"

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  asChild?: boolean
  variant?: "default" | "ghost" | "outline" | "danger"
  size?: "sm" | "md" | "icon"
}

export function Button({ className, asChild, variant = "default", size = "md", ...props }: ButtonProps) {
  const Comp = asChild ? Slot : "button"
  return (
    <Comp
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-md border text-sm font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand disabled:pointer-events-none disabled:opacity-50",
        variant === "default" && "border-brand bg-brand px-3 text-white shadow-panel hover:brightness-95",
        variant === "ghost" && "border-transparent bg-transparent px-2 hover:bg-muted",
        variant === "outline" && "border-border bg-panel px-3 hover:bg-muted",
        variant === "danger" && "border-danger bg-danger px-3 text-white hover:brightness-95",
        size === "sm" && "h-8",
        size === "md" && "h-9",
        size === "icon" && "h-9 w-9 px-0",
        className
      )}
      {...props}
    />
  )
}
