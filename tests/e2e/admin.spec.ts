import { expect, test } from "@playwright/test"

test("admin shell renders login and dashboard routes", async ({ page }) => {
  await page.goto("/admin/login")
  await expect(page.getByRole("heading", { name: /sign in/i })).toBeVisible()

  await page.goto("/admin/dashboard")
  await expect(page.getByRole("heading", { name: /dashboard/i })).toBeVisible()
})
