import type { Metadata } from "next"
import "./globals.css"
import { Providers } from "@/components/admin/providers"

export const metadata: Metadata = {
  title: "GoMyAdmin Demo",
  description: "Production-ready backoffice for Go applications"
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <Providers>{children}</Providers>
      </body>
    </html>
  )
}
