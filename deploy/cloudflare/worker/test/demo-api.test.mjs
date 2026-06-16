import assert from "node:assert/strict"
import test from "node:test"

import { createDemoWorker } from "../src/demo-api.mjs"

const worker = createDemoWorker()

async function json(method, path, body) {
  const response = await worker.fetch(
    new Request(`https://demo-api.example.com${path}`, {
      method,
      headers: {
        "content-type": "application/json",
        origin: "https://demo-ui.pages.dev"
      },
      body: body ? JSON.stringify(body) : undefined
    }),
    { ADMIN_ORIGIN: "https://demo-ui.pages.dev" }
  )
  return { response, payload: await response.json() }
}

test("preflight returns CORS headers for the Pages frontend", async () => {
  const response = await worker.fetch(
    new Request("https://demo-api.example.com/admin/api/resources", {
      method: "OPTIONS",
      headers: {
        origin: "https://demo-ui.pages.dev",
        "access-control-request-method": "GET"
      }
    }),
    { ADMIN_ORIGIN: "https://demo-ui.pages.dev" }
  )

  assert.equal(response.status, 204)
  assert.equal(response.headers.get("access-control-allow-origin"), "https://demo-ui.pages.dev")
  assert.equal(response.headers.get("access-control-allow-credentials"), "true")
  assert.match(response.headers.get("access-control-allow-methods") ?? "", /GET/)
})

test("login accepts the public demo credentials", async () => {
  const { response, payload } = await json("POST", "/admin/api/auth/login", {
    email: "admin@example.com",
    password: "password"
  })

  assert.equal(response.status, 200)
  assert.equal(payload.error, null)
  assert.equal(payload.data.user.email, "admin@example.com")
  assert.match(response.headers.get("set-cookie") ?? "", /gomyadmin_demo_session=/)
})

test("resources returns metadata expected by the frontend", async () => {
  const { response, payload } = await json("GET", "/admin/api/resources")

  assert.equal(response.status, 200)
  assert.equal(payload.error, null)
  assert.ok(payload.data.length >= 3)
  assert.ok(payload.data.some((resource) => resource.name === "customers"))
  assert.ok(payload.data.find((resource) => resource.name === "customers").fields.length > 0)
})

test("resource list supports search and pagination metadata", async () => {
  const { response, payload } = await json("GET", "/admin/api/customers?page=1&per_page=2&q=acme")

  assert.equal(response.status, 200)
  assert.equal(payload.error, null)
  assert.equal(payload.data.length, 1)
  assert.equal(payload.data[0].name, "Acme Robotics")
  assert.equal(payload.meta.page, 1)
  assert.equal(payload.meta.per_page, 2)
  assert.equal(payload.meta.total, 1)
})

test("resource export returns csv", async () => {
  const response = await worker.fetch(
    new Request("https://demo-api.example.com/admin/api/customers/export", {
      headers: { origin: "https://demo-ui.pages.dev" }
    }),
    { ADMIN_ORIGIN: "https://demo-ui.pages.dev" }
  )
  const body = await response.text()

  assert.equal(response.status, 200)
  assert.match(response.headers.get("content-type") ?? "", /text\/csv/)
  assert.match(body, /id,name,email,status,plan/)
  assert.match(body, /Acme Robotics/)
})

test("resource detail, create, update, and delete return success envelopes", async () => {
  const detail = await json("GET", "/admin/api/customers/cus_001")
  assert.equal(detail.response.status, 200)
  assert.equal(detail.payload.data.id, "cus_001")

  const created = await json("POST", "/admin/api/customers", {
    name: "New Customer",
    email: "new@example.com",
    status: "lead",
    plan: "starter"
  })
  assert.equal(created.response.status, 201)
  assert.equal(created.payload.data.name, "New Customer")

  const updated = await json("PATCH", "/admin/api/customers/cus_001", { status: "active" })
  assert.equal(updated.response.status, 200)
  assert.equal(updated.payload.data.status, "active")

  const deleted = await json("DELETE", "/admin/api/customers/cus_001")
  assert.equal(deleted.response.status, 200)
  assert.equal(deleted.payload.data.deleted, true)
})
