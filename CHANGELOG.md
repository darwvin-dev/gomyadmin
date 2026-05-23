# Changelog

All notable changes to GoMyAdmin are documented here.  
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [0.1.0] — 2026-05-23

Initial public release.

### Added

**Core resource API (`pkg/admin`)**
- `App` — thread-safe resource registry with stable name ordering
- `Resource` builder — label, plural label, icon, description, table, primary key, default sort, page size, navigation group, visibility rule
- `Field` builder — 25 field types: string, text, email, password, number, integer, float, decimal, money, boolean, enum, status badge, date, datetime, time, uuid, json, jsonb, markdown, rich_text, image, file, relation, computed
- Field flags — required, unique, nullable, searchable, sortable, filterable, readonly, hidden, hidden_in_list, hidden_in_form, hidden_in_detail, primary, tenant_key
- Relation support — `BelongsTo`, `HasMany`, `ForeignKey`, `Display`
- `Action` builder — description, icon, danger, requires_confirmation, requires_reason, input schema, permission, timeout
- `Policy` interface — `CanView`, `CanCreate`, `CanUpdate`, `CanDelete`; `AllowAllPolicy`, `DenyAllPolicy`
- `Context` and `Actor` — permission checking with `*`, `resource.*`, `*.operation` wildcard patterns
- `WriteJSON` / `WriteError` — uniform JSON response envelope

**CLI (`cmd/gomyadmin`)**
- `init` — scaffold Go + PostgreSQL + Next.js project
- `generate resource` — emit resource metadata file from template
- `introspect` — dump PostgreSQL schema to JSON
- `serve`, `dev`, `build`, `doctor`, `demo`, `openapi generate`, `version`

**PostgreSQL introspection (`internal/introspect`)**
- Full schema dump via `information_schema` — tables, columns, primary keys
- Column-to-field-type mapping: `ColumnToFieldType`, `SuggestFieldType`
- Heuristic helpers: `IsLikelyPrimaryKey`, `IsLikelyEmail`, `IsLikelyPassword`, `IsSearchable`, `IsSortable`

**Code generation (`internal/generator`)**
- `InitProject` — full project scaffold with atomic rollback on error
- `GenerateResource` — `gofmt`-formatted resource file; rejects conflicts unless `--force`

**Authentication (`pkg/auth`)**
- Argon2id password hashing with configurable memory, iterations, parallelism
- Memory session store + `SessionManager` — secure cookie, configurable TTL, `SameSite=Lax`
- CSRF double-submit cookie — `CSRFMiddleware`, `IssueCSRF`
- Sliding window rate limiter per IP — `X-Forwarded-For` aware

**Authorization (`pkg/rbac`)**
- Five built-in roles: `super_admin`, `tenant_admin`, `manager`, `support`, `viewer`
- Wildcard permission matching — `*`, `resource.*`, `*.operation`

**Audit log (`pkg/audit`)**
- Structured `Event` with actor, tenant, action, resource, old/new values, IP, request ID
- Thread-safe `MemoryStore` with filtering by actor, tenant, resource, action, date range

**File storage (`pkg/storage`)**
- `Storage` and `Inspector` interfaces
- `Local` adapter — path traversal protection, public URL, signed URL with expiry
- `Memory` adapter — thread-safe in-memory store for tests

**Database (`pkg/postgres`)**
- pgxpool `Connect` with configurable pool settings and statement timeout
- `QueryBuilder` — safe SELECT and COUNT; all user values parameterized; tenant scoping, ILIKE search, operator validation

**Other packages**
- `pkg/pagination` — `page` / `per_page` parsing, capped at `MaxPerPage=100`
- `pkg/filters` — search, sort, filter query parsing with operator validation and deterministic ordering
- `pkg/tenant` — `ActorResolver`, `StaticResolver`
- `pkg/openapi` — OpenAPI 3.1 spec generation for all resource endpoints and actions
- `pkg/logger` — slog-based JSON logger

**Templates and examples**
- `templates/backend-go` — runnable Go admin API server
- `templates/frontend-next-shadcn` — Next.js 16 + TypeScript + Tailwind + shadcn/ui
- `examples/basic`, `examples/crm`, `examples/saas`, `examples/sms-gateway`

**Infrastructure**
- Docker Compose with PostgreSQL 17, Go backend, Next.js frontend
- GitHub Actions CI — Go test + vet, frontend typecheck + build
- Comprehensive documentation in `docs/`

---

[0.1.0]: https://github.com/darwvin/gomyadmin/releases/tag/v0.1.0
