import { defineConfig, devices } from "@playwright/test"

export default defineConfig({
  testDir: ".",
  outputDir: "test-results",
  timeout: 30_000,
  use: {
    baseURL: process.env.GOMYADMIN_E2E_BASE_URL ?? "http://localhost:3000",
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    video: "retain-on-failure"
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] }
    }
  ]
})
