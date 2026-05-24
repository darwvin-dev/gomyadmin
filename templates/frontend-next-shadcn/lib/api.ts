export type ApiResponse<T> = {
  data: T | null
  meta: Record<string, unknown>
  error: null | {
    code: string
    message: string
    fields?: Record<string, string[]>
  }
}

export type FieldMeta = {
  name: string
  label: string
  type: string
  searchable: boolean
  sortable: boolean
  filterable: boolean
  readonly: boolean
  hidden: boolean
  enum_values?: string[]
}

export type ActionMeta = {
  name: string
  label: string
  icon: string
  danger: boolean
  requires_confirmation: boolean
  requires_reason: boolean
}

export type ResourceMeta = {
  name: string
  label: string
  icon: string
  description: string
  fields: FieldMeta[]
  actions: ActionMeta[]
}

export type RecordRow = Record<string, unknown>

const baseURL = process.env.NEXT_PUBLIC_ADMIN_API_URL ?? "http://localhost:8080"

async function request<T>(path: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
  const response = await fetch(`${baseURL}${path}`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(options.headers ?? {})
    },
    ...options
  })
  const payload = (await response.json()) as ApiResponse<T>
  if (!response.ok || payload.error) {
    throw new Error(payload.error?.message ?? "Request failed")
  }
  return payload
}

export const api = {
  login(email: string, password: string) {
    return request<{ user: RecordRow }>("/admin/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password })
    })
  },
  me() {
    return request<{ user: RecordRow; tenants: RecordRow[] }>("/admin/api/me")
  },
  resources() {
    return request<ResourceMeta[]>("/admin/api/resources")
  },
  list(resource: string, params: URLSearchParams) {
    return request<RecordRow[]>(`/admin/api/${resource}?${params.toString()}`)
  },
  get(resource: string, id: string) {
    return request<RecordRow>(`/admin/api/${resource}/${id}`)
  },
  create(resource: string, payload: RecordRow) {
    return request<RecordRow>(`/admin/api/${resource}`, { method: "POST", body: JSON.stringify(payload) })
  },
  update(resource: string, id: string, payload: RecordRow) {
    return request<RecordRow>(`/admin/api/${resource}/${id}`, { method: "PATCH", body: JSON.stringify(payload) })
  },
  remove(resource: string, id: string) {
    return request<{ deleted: boolean }>(`/admin/api/${resource}/${id}`, { method: "DELETE" })
  },
  bulkDelete(resource: string, ids: string[]) {
    return request<{ deleted: number; ids: string[] }>(`/admin/api/${resource}/bulk-delete`, {
      method: "POST",
      body: JSON.stringify({ ids })
    })
  },
  action(resource: string, id: string, action: string, payload: RecordRow) {
    return request<{ message: string }>(`/admin/api/${resource}/${id}/actions/${action}`, {
      method: "POST",
      body: JSON.stringify(payload)
    })
  },
  audit() {
    return request<RecordRow[]>("/admin/api/audit")
  },
  files() {
    return request<RecordRow[]>("/admin/api/files")
  }
}
