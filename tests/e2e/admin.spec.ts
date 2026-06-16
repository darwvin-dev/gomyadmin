import { expect, test, type Page, type Route } from "@playwright/test"

type ApiPayload = Record<string, unknown>
type UserRecord = {
  id: string
  email: string
  name: string
  role: string
  status: string
  created_at: string
}
type AuditRecord = {
  id: string
  actor_email: string
  action: string
  resource: string
  old_values: ApiPayload | null
  new_values: ApiPayload | null
  metadata: ApiPayload
  created_at: string
}

const resources = [
  {
    name: "users",
    label: "Users",
    icon: "users",
    description: "Administrators and operators who can access the workspace.",
    actions: [],
    fields: [
      { name: "id", label: "ID", type: "string", searchable: false, sortable: false, filterable: false, readonly: true, hidden: true },
      { name: "email", label: "Email", type: "email", searchable: true, sortable: true, filterable: false, readonly: false, hidden: false },
      { name: "name", label: "Name", type: "string", searchable: true, sortable: true, filterable: false, readonly: false, hidden: false },
      { name: "role", label: "Role", type: "enum", searchable: false, sortable: false, filterable: true, readonly: false, hidden: false, enum_values: ["admin", "manager", "support", "viewer"] },
      { name: "status", label: "Status", type: "enum", searchable: false, sortable: false, filterable: true, readonly: false, hidden: false, enum_values: ["active", "blocked", "pending"] },
      { name: "created_at", label: "Created", type: "datetime", searchable: false, sortable: true, filterable: false, readonly: true, hidden: false }
    ]
  }
]

function apiResponse<T>(data: T, meta: ApiPayload = {}, error: null | { code: string; message: string } = null) {
  return {
    data,
    meta,
    error
  }
}

async function fulfill(route: Route, data: unknown, meta?: ApiPayload) {
  await route.fulfill({
    status: 200,
    contentType: "application/json",
    body: JSON.stringify(apiResponse(data, meta))
  })
}

async function mockAdminAPI(page: Page) {
  const users: UserRecord[] = [
    {
      id: "usr_1",
      email: "avery@example.com",
      name: "Avery Stone",
      role: "admin",
      status: "active",
      created_at: "2026-01-05T10:00:00Z"
    },
    {
      id: "usr_2",
      email: "blake@example.com",
      name: "Blake Wong",
      role: "support",
      status: "blocked",
      created_at: "2026-01-04T10:00:00Z"
    }
  ]
  const audit: AuditRecord[] = [
    {
      id: "evt_1",
      actor_email: "admin@example.com",
      action: "user.created",
      resource: "users",
      old_values: null,
      new_values: { id: "usr_2", email: "blake@example.com" },
      metadata: { ip: "127.0.0.1" },
      created_at: "2026-01-05T11:00:00Z"
    }
  ]

  await page.route("**/admin/api/**", async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = request.method()
    const path = url.pathname

    if (path === "/admin/api/auth/providers" && method === "GET") return fulfill(route, [])
    if (path === "/admin/api/auth/login" && method === "POST") return fulfill(route, { user: { id: "admin_1", email: "admin@example.com" } })
    if (path === "/admin/api/resources" && method === "GET") return fulfill(route, resources)
    if (path === "/admin/api/audit" && method === "GET") return fulfill(route, audit)

    if (path === "/admin/api/users" && method === "GET") {
      const search = url.searchParams.get("q")?.toLowerCase()
      const status = url.searchParams.get("filter[status][eq]")
      const filtered = users.filter((user) => {
        const matchesSearch = search ? user.name.toLowerCase().includes(search) || user.email.toLowerCase().includes(search) : true
        const matchesStatus = status ? user.status === status : true
        return matchesSearch && matchesStatus
      })
      return fulfill(route, filtered, { total: filtered.length })
    }

    if (path === "/admin/api/users" && method === "POST") {
      const payload = request.postDataJSON() as ApiPayload
      const created: UserRecord = {
        id: "usr_3",
        email: String(payload.email ?? ""),
        name: String(payload.name ?? ""),
        role: String(payload.role ?? ""),
        status: String(payload.status ?? ""),
        created_at: "2026-01-06T10:00:00Z"
      }
      users.unshift(created)
      audit.unshift({
        id: "evt_2",
        actor_email: "admin@example.com",
        action: "user.created",
        resource: "users",
        old_values: null,
        new_values: created,
        metadata: { source: "e2e" },
        created_at: "2026-01-06T10:01:00Z"
      })
      return fulfill(route, created)
    }

    const userPath = path.match(/^\/admin\/api\/users\/([^/]+)$/)
    if (userPath && method === "GET") {
      const user = users.find((item) => item.id === userPath[1])
      return user ? fulfill(route, user) : route.fulfill({ status: 404, contentType: "application/json", body: JSON.stringify(apiResponse(null, {}, { code: "not_found", message: "User not found" })) })
    }

    if (userPath && method === "PATCH") {
      const payload = request.postDataJSON() as ApiPayload
      const index = users.findIndex((item) => item.id === userPath[1])
      if (index === -1) return route.fulfill({ status: 404, contentType: "application/json", body: JSON.stringify(apiResponse(null, {}, { code: "not_found", message: "User not found" })) })
      const previous = users[index]
      const updated: UserRecord = {
        ...previous,
        email: typeof payload.email === "string" ? payload.email : previous.email,
        name: typeof payload.name === "string" ? payload.name : previous.name,
        role: typeof payload.role === "string" ? payload.role : previous.role,
        status: typeof payload.status === "string" ? payload.status : previous.status
      }
      users[index] = updated
      audit.unshift({
        id: "evt_3",
        actor_email: "admin@example.com",
        action: "user.updated",
        resource: "users",
        old_values: { id: updated.id },
        new_values: updated,
        metadata: { source: "e2e" },
        created_at: "2026-01-06T10:02:00Z"
      })
      return fulfill(route, updated)
    }

    return route.fallback()
  })
}

test.beforeEach(async ({ page }) => {
  await mockAdminAPI(page)
})

test("logs in and loads the CRM resource list", async ({ page }) => {
  await page.goto("/admin/login")
  await expect(page.getByRole("heading", { name: "GoMyAdmin" })).toBeVisible()

  await page.getByLabel("Email").fill("admin@example.com")
  await page.getByLabel("Password").fill("password")
  await page.getByRole("button", { name: /continue/i }).click()

  await expect(page).toHaveURL(/\/admin\/dashboard$/)
  await page.goto("/admin/resources/users")

  await expect(page.getByRole("heading", { name: "Users" })).toBeVisible()
  await expect(page.getByRole("cell", { name: "avery@example.com" })).toBeVisible()
  await expect(page.getByText("Page 1 of 1 · 2 records")).toBeVisible()
})

test("searches and filters the resource list", async ({ page }) => {
  await page.goto("/admin/resources/users")

  await page.getByPlaceholder("Search users").fill("avery")
  await expect(page.getByRole("cell", { name: "avery@example.com" })).toBeVisible()
  await expect(page.getByRole("cell", { name: "blake@example.com" })).toHaveCount(0)
  await expect(page.getByText("Page 1 of 1 · 1 records")).toBeVisible()

  await page.getByPlaceholder("Search users").fill("")
  await page.getByLabel("Clear filter").click()
  await page.getByRole("combobox").first().selectOption("status")
  await page.getByPlaceholder("Value").fill("blocked")
  await page.getByRole("button", { name: "Apply" }).click()

  await expect(page.getByRole("cell", { name: "blake@example.com" })).toBeVisible()
  await expect(page.getByRole("cell", { name: "avery@example.com" })).toHaveCount(0)
})

test("creates and updates a CRM record", async ({ page }) => {
  await page.goto("/admin/resources/users/new")

  await expect(page.getByRole("heading", { name: "New Users" })).toBeVisible()
  await page.getByLabel("Email").fill("casey@example.com")
  await page.getByLabel("Name").fill("Casey Admin")
  await page.getByLabel("Role").selectOption("manager")
  await page.getByLabel("Status").selectOption("pending")
  await page.getByRole("button", { name: "Save" }).click()

  await expect(page).toHaveURL(/\/admin\/resources\/users$/)
  await expect(page.getByRole("cell", { name: "casey@example.com" })).toBeVisible()

  await page.goto("/admin/resources/users/usr_3/edit")
  await expect(page.getByRole("heading", { name: "Edit Users" })).toBeVisible()
  await page.getByLabel("Name").fill("Casey Operator")
  await page.getByLabel("Status").selectOption("active")
  await page.getByRole("button", { name: "Save" }).click()

  await expect(page).toHaveURL(/\/admin\/resources\/users$/)
  await expect(page.getByRole("cell", { name: "Casey Operator" })).toBeVisible()
})

test("shows resource mutations in the audit log", async ({ page }) => {
  await page.goto("/admin/resources/users/new")
  await page.getByLabel("Email").fill("delta@example.com")
  await page.getByLabel("Name").fill("Delta Support")
  await page.getByLabel("Role").selectOption("support")
  await page.getByLabel("Status").selectOption("active")
  await page.getByRole("button", { name: "Save" }).click()

  await page.goto("/admin/audit")

  await expect(page.getByRole("heading", { name: "Audit log" })).toBeVisible()
  await expect(page.getByText("user.created · users").first()).toBeVisible()
  await expect(page.getByText("delta@example.com")).toBeVisible()
})
