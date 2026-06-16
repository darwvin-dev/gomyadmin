const now = "2026-06-12T10:00:00Z"

const resources = [
  {
    name: "users",
    label: "Users",
    icon: "users",
    description: "CRM operators and support teammates",
    fields: [
      field("id", "ID", "uuid", { sortable: true, readonly: true }),
      field("email", "Email", "email", { searchable: true, sortable: true }),
      field("name", "Name", "string", { searchable: true, sortable: true }),
      field("role", "Role", "enum", { filterable: true, enum_values: ["super_admin", "tenant_admin", "manager", "support", "viewer"] }),
      field("status", "Status", "status", { filterable: true, enum_values: ["active", "blocked", "pending"] }),
      field("tenant_id", "Tenant", "uuid", { hidden: true }),
      field("created_at", "Created", "datetime", { sortable: true, filterable: true, readonly: true }),
      field("updated_at", "Updated", "datetime", { sortable: true, readonly: true })
    ],
    actions: [
      action("block-user", "Block User", "ban", { danger: true, requires_confirmation: true, requires_reason: true }),
      action("reset-password", "Reset Password", "key-round", { requires_confirmation: true })
    ]
  },
  {
    name: "customers",
    label: "Customers",
    icon: "building-2",
    description: "Companies and buyers managed by the CRM",
    fields: [
      field("id", "ID", "uuid", { sortable: true, readonly: true }),
      field("name", "Name", "string", { searchable: true, sortable: true }),
      field("email", "Email", "email", { searchable: true, sortable: true }),
      field("status", "Status", "status", { filterable: true, enum_values: ["active", "lead", "churned"] }),
      field("plan", "Plan", "enum", { filterable: true, enum_values: ["starter", "growth", "enterprise"] }),
      field("tenant_id", "Tenant", "uuid", { hidden: true }),
      field("created_at", "Created", "datetime", { sortable: true, filterable: true, readonly: true }),
      field("updated_at", "Updated", "datetime", { sortable: true, readonly: true })
    ],
    actions: []
  },
  {
    name: "invoices",
    label: "Invoices",
    icon: "receipt",
    description: "Billing records, payment state, and refund actions",
    fields: [
      field("id", "ID", "uuid", { sortable: true, readonly: true }),
      field("customer_id", "Customer", "relation", { filterable: true, relation: { resource: "customers", foreign_key: "customer_id", display_field: "name", kind: "belongs_to" } }),
      field("number", "Number", "string", { searchable: true, sortable: true }),
      field("amount", "Amount", "money", { sortable: true }),
      field("status", "Status", "status", { filterable: true, enum_values: ["draft", "open", "paid", "failed", "refunded"] }),
      field("tenant_id", "Tenant", "uuid", { hidden: true }),
      field("created_at", "Created", "datetime", { sortable: true, filterable: true, readonly: true }),
      field("updated_at", "Updated", "datetime", { sortable: true, readonly: true })
    ],
    actions: [
      action("mark-as-paid", "Mark as Paid", "check-circle", { requires_confirmation: true }),
      action("refund-invoice", "Refund Invoice", "undo-2", { danger: true, requires_confirmation: true, requires_reason: true })
    ]
  },
  {
    name: "tickets",
    label: "Tickets",
    icon: "life-buoy",
    description: "Customer support queue and escalation workflow",
    fields: [
      field("id", "ID", "uuid", { sortable: true, readonly: true }),
      field("customer_id", "Customer", "relation", { filterable: true, relation: { resource: "customers", foreign_key: "customer_id", display_field: "name", kind: "belongs_to" } }),
      field("subject", "Subject", "string", { searchable: true, sortable: true }),
      field("priority", "Priority", "enum", { filterable: true, enum_values: ["low", "normal", "high", "urgent"] }),
      field("status", "Status", "status", { filterable: true, enum_values: ["open", "waiting", "solved"] }),
      field("tenant_id", "Tenant", "uuid", { hidden: true }),
      field("created_at", "Created", "datetime", { sortable: true, filterable: true, readonly: true }),
      field("updated_at", "Updated", "datetime", { sortable: true, readonly: true })
    ],
    actions: [action("close-ticket", "Close Ticket", "check", { requires_confirmation: true })]
  }
]

const rows = {
  users: [
    record({ id: "usr_001", email: "admin@example.com", name: "Demo Admin", role: "super_admin", status: "active" }),
    record({ id: "usr_002", email: "support@example.com", name: "Support Lead", role: "support", status: "active" })
  ],
  customers: [
    record({ id: "cus_001", name: "Acme Robotics", email: "ops@acme.example", status: "active", plan: "enterprise" }),
    record({ id: "cus_002", name: "Northstar Labs", email: "billing@northstar.example", status: "lead", plan: "growth" }),
    record({ id: "cus_003", name: "Kite Systems", email: "hello@kite.example", status: "churned", plan: "starter" })
  ],
  invoices: [
    record({ id: "inv_001", customer_id: "cus_001", number: "INV-1001", amount: 12900, status: "paid" }),
    record({ id: "inv_002", customer_id: "cus_002", number: "INV-1002", amount: 4900, status: "open" }),
    record({ id: "inv_003", customer_id: "cus_003", number: "INV-1003", amount: 900, status: "failed" })
  ],
  tickets: [
    record({ id: "tic_001", customer_id: "cus_001", subject: "Invoice export question", priority: "normal", status: "open" }),
    record({ id: "tic_002", customer_id: "cus_002", subject: "Upgrade request", priority: "high", status: "waiting" })
  ],
  audit: [
    {
      id: "aud_001",
      actor_id: "usr_001",
      actor_email: "admin@example.com",
      tenant_id: "tenant_demo",
      action: "login",
      resource: "auth",
      resource_id: "usr_001",
      ip_address: "203.0.113.10",
      user_agent: "Cloudflare demo",
      request_id: "demo-request-1",
      created_at: now
    }
  ],
  files: [
    {
      id: "file_001",
      name: "customer-import.csv",
      content_type: "text/csv",
      size: 18240,
      visibility: "private",
      tenant_id: "tenant_demo",
      created_at: now
    }
  ]
}

export function createDemoWorker() {
  return {
    async fetch(request, env = {}) {
      return handleRequest(request, env)
    }
  }
}

async function handleRequest(request, env) {
  const url = new URL(request.url)
  const cors = corsHeaders(request, env)

  if (request.method === "OPTIONS") {
    return new Response(null, { status: 204, headers: cors })
  }

  try {
    if (request.method === "POST" && url.pathname === "/admin/api/auth/login") {
      const input = await readJSON(request)
      if (input.email !== "admin@example.com" || input.password !== "password") {
        return error("INVALID_CREDENTIALS", "Invalid email or password", 401, cors)
      }
      return json(
        {
          user: demoActor(),
          expires_at: new Date(Date.now() + 8 * 60 * 60 * 1000).toISOString()
        },
        {},
        200,
        {
          ...cors,
          "set-cookie": "gomyadmin_demo_session=demo; Path=/; HttpOnly; Secure; SameSite=None; Max-Age=28800"
        }
      )
    }

    if (request.method === "GET" && url.pathname === "/admin/api/auth/providers") return json([], {}, 200, cors)
    if (request.method === "POST" && url.pathname === "/admin/api/auth/logout") return json({ ok: true }, {}, 200, cors)
    if (request.method === "GET" && url.pathname === "/admin/api/me") {
      return json({ user: demoActor(), tenants: [{ id: "tenant_demo", name: "Demo Workspace", slug: "demo" }] }, {}, 200, cors)
    }
    if (request.method === "GET" && url.pathname === "/admin/api/auth/api-keys") return json([], {}, 200, cors)
    if (request.method === "POST" && url.pathname === "/admin/api/auth/api-keys") {
      return json({ key: { id: "key_demo", name: "Demo key", prefix: "gma_demo", scopes: ["read"], created_at: now }, secret: "gma_demo_not_for_production" }, {}, 201, cors)
    }
    if (request.method === "POST" && url.pathname.match(/^\/admin\/api\/auth\/api-keys\/[^/]+\/revoke$/)) {
      return json({ ok: true }, {}, 200, cors)
    }
    if (request.method === "GET" && url.pathname === "/admin/api/resources") return json(resources, {}, 200, cors)
    if (request.method === "GET" && url.pathname === "/admin/api/audit") return json(rows.audit, { total: rows.audit.length }, 200, cors)
    if (request.method === "GET" && url.pathname === "/admin/api/files") return json(rows.files, { total: rows.files.length }, 200, cors)

    const route = parseResourceRoute(url.pathname)
    if (!route) return error("NOT_FOUND", "Route not found", 404, cors)
    if (!resourceByName(route.resource)) return error("NOT_FOUND", "Resource not found", 404, cors)

    if (request.method === "GET" && route.export) return csv(route.resource, cors)
    if (request.method === "GET" && !route.id) return list(route.resource, url.searchParams, cors)
    if (request.method === "GET" && route.id) return detail(route.resource, route.id, cors)
    if (request.method === "POST" && route.bulkDelete) return json({ deleted: 0, ids: [] }, {}, 200, cors)
    if (request.method === "POST" && route.action) return json({ message: "Demo action completed" }, {}, 200, cors)
    if (request.method === "POST" && !route.id) return create(route.resource, await readJSON(request), cors)
    if (request.method === "PATCH" && route.id) return update(route.resource, route.id, await readJSON(request), cors)
    if (request.method === "DELETE" && route.id) return json({ deleted: true }, {}, 200, cors)

    return error("METHOD_NOT_ALLOWED", "Method not allowed", 405, cors)
  } catch (err) {
    return error("DEMO_API_ERROR", err instanceof Error ? err.message : "Demo API failed", 500, cors)
  }
}

function parseResourceRoute(pathname) {
  const match = pathname.match(/^\/admin\/api\/([^/]+)(?:\/([^/]+))?(?:\/([^/]+))?/)
  if (!match) return null
  return {
    resource: match[1],
    id: match[2] && !["export", "bulk-delete", "bulk-actions"].includes(match[2]) ? match[2] : "",
    export: match[2] === "export",
    bulkDelete: match[2] === "bulk-delete",
    action: match[2] && match[3] === "actions"
  }
}

function list(resource, params, headers) {
  let data = [...(rows[resource] ?? [])]
  const query = (params.get("q") ?? "").toLowerCase().trim()
  if (query) {
    data = data.filter((row) => Object.values(row).some((value) => String(value ?? "").toLowerCase().includes(query)))
  }

  for (const [key, value] of params.entries()) {
    const filterMatch = key.match(/^filter\[([^\]]+)\]\[([^\]]+)\]$/)
    if (!filterMatch || value === "") continue
    const [, fieldName, operator] = filterMatch
    data = data.filter((row) => compareFilter(row[fieldName], operator, value))
  }

  const sort = params.get("sort") ?? "-created_at"
  const desc = sort.startsWith("-")
  const sortField = desc ? sort.slice(1) : sort
  data.sort((left, right) => String(left[sortField] ?? "").localeCompare(String(right[sortField] ?? "")) * (desc ? -1 : 1))

  const page = Math.max(1, Number(params.get("page") ?? "1"))
  const perPage = Math.max(1, Number(params.get("per_page") ?? "25"))
  const total = data.length
  const start = (page - 1) * perPage
  return json(data.slice(start, start + perPage), { page, per_page: perPage, total }, 200, headers)
}

function detail(resource, id, headers) {
  const row = (rows[resource] ?? []).find((item) => item.id === id)
  if (!row) return error("NOT_FOUND", "Record not found", 404, headers)
  return json(row, {}, 200, headers)
}

function create(resource, input, headers) {
  return json(record({ id: `${resource.slice(0, 3)}_${Date.now()}`, ...input }), {}, 201, headers)
}

function update(resource, id, input, headers) {
  const existing = (rows[resource] ?? []).find((item) => item.id === id) ?? record({ id })
  return json({ ...existing, ...input, id, updated_at: now }, {}, 200, headers)
}

function csv(resource, headers) {
  const resourceMeta = resourceByName(resource)
  const fields = resourceMeta.fields.filter((item) => !item.hidden).map((item) => item.name)
  const body = [fields.join(","), ...(rows[resource] ?? []).map((row) => fields.map((key) => csvCell(row[key])).join(","))].join("\n")
  return new Response(body + "\n", {
    status: 200,
    headers: {
      ...headers,
      "content-type": "text/csv; charset=utf-8",
      "content-disposition": `attachment; filename="${resource}.csv"`
    }
  })
}

function resourceByName(name) {
  return resources.find((resource) => resource.name === name)
}

function compareFilter(value, operator, expected) {
  const actual = String(value ?? "").toLowerCase()
  const target = String(expected ?? "").toLowerCase()
  if (operator === "contains") return actual.includes(target)
  if (operator === "starts_with") return actual.startsWith(target)
  if (operator === "ends_with") return actual.endsWith(target)
  if (operator === "gte") return Number(value) >= Number(expected)
  if (operator === "lte") return Number(value) <= Number(expected)
  return actual === target
}

function corsHeaders(request, env) {
  const origin = request.headers.get("origin") ?? ""
  const allowed = env.ADMIN_ORIGIN || origin || "*"
  return {
    "access-control-allow-origin": allowed === "*" ? "*" : origin || allowed,
    "access-control-allow-credentials": "true",
    "access-control-allow-methods": "GET,POST,PATCH,DELETE,OPTIONS",
    "access-control-allow-headers": "content-type,authorization,x-api-key,x-csrf-token",
    "access-control-max-age": "86400",
    vary: "Origin"
  }
}

function json(data, meta = {}, status = 200, headers = {}) {
  return new Response(JSON.stringify({ data, meta, error: null }), {
    status,
    headers: {
      ...headers,
      "content-type": "application/json; charset=utf-8"
    }
  })
}

function error(code, message, status, headers = {}) {
  return new Response(JSON.stringify({ data: null, meta: {}, error: { code, message } }), {
    status,
    headers: {
      ...headers,
      "content-type": "application/json; charset=utf-8"
    }
  })
}

async function readJSON(request) {
  if (!request.body) return {}
  return request.json()
}

function demoActor() {
  return {
    id: "usr_001",
    email: "admin@example.com",
    name: "Demo Admin",
    role: "super_admin",
    tenant_id: "tenant_demo"
  }
}

function field(name, label, type, options = {}) {
  return {
    name,
    label,
    type,
    searchable: Boolean(options.searchable),
    sortable: Boolean(options.sortable),
    filterable: Boolean(options.filterable),
    readonly: Boolean(options.readonly),
    hidden: Boolean(options.hidden),
    ...(options.enum_values ? { enum_values: options.enum_values } : {}),
    ...(options.relation ? { relation: options.relation } : {})
  }
}

function action(name, label, icon, options = {}) {
  return {
    name,
    label,
    icon,
    danger: Boolean(options.danger),
    requires_confirmation: Boolean(options.requires_confirmation),
    requires_reason: Boolean(options.requires_reason)
  }
}

function record(input) {
  return {
    tenant_id: "tenant_demo",
    created_at: now,
    updated_at: now,
    ...input
  }
}

function csvCell(value) {
  const text = String(value ?? "")
  if (!/[",\n]/.test(text)) return text
  return `"${text.replaceAll('"', '""')}"`
}
