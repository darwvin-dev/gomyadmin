import { expect, test, type Page, type Route } from "@playwright/test"

type ApiPayload = Record<string, unknown>

// A target resource (companies) and a resource that belongs to it (projects).
const resources = [
  {
    name: "companies",
    label: "Companies",
    icon: "building",
    description: "Organizations that own projects.",
    actions: [],
    fields: [
      { name: "id", label: "ID", type: "string", searchable: false, sortable: false, filterable: false, readonly: true, hidden: true },
      { name: "name", label: "Name", type: "string", searchable: true, sortable: true, filterable: false, readonly: false, hidden: false }
    ]
  },
  {
    name: "projects",
    label: "Projects",
    icon: "folder",
    description: "Delivery projects owned by a company.",
    actions: [],
    fields: [
      { name: "id", label: "ID", type: "string", searchable: false, sortable: false, filterable: false, readonly: true, hidden: true },
      { name: "name", label: "Name", type: "string", searchable: true, sortable: true, filterable: false, readonly: false, hidden: false },
      {
        name: "company_id",
        label: "Company",
        type: "relation",
        searchable: false,
        sortable: false,
        filterable: true,
        readonly: false,
        hidden: false,
        relation: { resource: "companies", foreign_key: "company_id", display_field: "name", kind: "belongs_to" }
      }
    ]
  }
]

function apiResponse<T>(data: T, meta: ApiPayload = {}) {
  return { data, meta, error: null }
}

async function fulfill(route: Route, data: unknown, meta?: ApiPayload) {
  await route.fulfill({
    status: 200,
    contentType: "application/json",
    body: JSON.stringify(apiResponse(data, meta))
  })
}

async function mockAdminAPI(page: Page) {
  const companies = [
    { id: "co_1", name: "Acme Robotics" },
    { id: "co_2", name: "Globex" }
  ]
  const projects = [{ id: "prj_1", name: "Apollo", company_id: "co_1" }]

  await page.route("**/admin/api/**", async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = request.method()
    const path = url.pathname

    if (path === "/admin/api/auth/providers" && method === "GET") return fulfill(route, [])
    if (path === "/admin/api/auth/login" && method === "POST") return fulfill(route, { user: { id: "admin_1", email: "admin@example.com" } })
    if (path === "/admin/api/resources" && method === "GET") return fulfill(route, resources)
    if (path === "/admin/api/audit" && method === "GET") return fulfill(route, [])

    if (path === "/admin/api/companies" && method === "GET") return fulfill(route, companies, { total: companies.length })
    if (path === "/admin/api/projects" && method === "GET") return fulfill(route, projects, { total: projects.length })

    const projectPath = path.match(/^\/admin\/api\/projects\/([^/]+)$/)
    if (projectPath && method === "GET") {
      const project = projects.find((item) => item.id === projectPath[1])
      return fulfill(route, project ?? null)
    }
    if (projectPath && method === "PATCH") {
      const payload = request.postDataJSON() as ApiPayload
      const index = projects.findIndex((item) => item.id === projectPath[1])
      if (index !== -1) projects[index] = { ...projects[index], ...payload } as (typeof projects)[number]
      return fulfill(route, projects[index])
    }

    return route.fallback()
  })
}

test.beforeEach(async ({ page }) => {
  await mockAdminAPI(page)
})

test("relation field renders the related label, not the raw id, in the list", async ({ page }) => {
  await page.goto("/admin/resources/projects")

  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible()
  // The Company column shows the related company name, never the foreign-key id.
  await expect(page.getByRole("cell", { name: "Acme Robotics" })).toBeVisible()
  await expect(page.getByRole("cell", { name: "co_1" })).toHaveCount(0)
})

test("relation field is editable through a selector and saves the related id", async ({ page }) => {
  await page.goto("/admin/resources/projects/prj_1/edit")

  await expect(page.getByRole("heading", { name: "Edit Projects" })).toBeVisible()

  // The Company field is a select populated with the related records' labels.
  const companySelect = page.getByLabel("Company")
  await expect(companySelect).toBeVisible()
  await expect(companySelect.getByRole("option", { name: "Acme Robotics" })).toHaveCount(1)
  await expect(companySelect.getByRole("option", { name: "Globex" })).toHaveCount(1)

  // Selecting a different company submits its id.
  const patch = page.waitForRequest((request) => request.url().includes("/admin/api/projects/prj_1") && request.method() === "PATCH")
  await companySelect.selectOption("co_2")
  await page.getByRole("button", { name: "Save" }).click()
  const body = (await patch).postDataJSON() as ApiPayload
  expect(body.company_id).toBe("co_2")
})
