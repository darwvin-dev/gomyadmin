# Changelog

All notable changes to GoMyAdmin are documented here.  
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [0.5.0] — 2026-06-02

### Added

- Integration test suite in `tests/integration/` (`//go:build integration`) for `pkg/server` (full CRUD login→logout), `pkg/migrate` (idempotency, checksum validation), and CLI binary (version, doctor, init, introspect)
- CI integration test step running against the existing PostgreSQL service
- CI coverage threshold: PRs fail if total coverage drops below 90%

### Changed

- `pkg/server/handler.go` split into focused files: `handler_auth.go`, `handler_crud.go`, `handler_files.go`, `handler_actions.go`, `handler_audit.go`, `handler_middleware.go`
- `internal/introspect`: introduced `querier` interface and `loadSchema` for testability without live DB
- `pkg/migrate` error messages now include migration version number

### Fixed

- **Breaking:** `newID()` in `pkg/adapters/sqlstore` now uses `crypto/rand` instead of `time.Now().UnixNano()` — IDs are now 16-character hex strings; eliminates collision risk under rapid record creation
- `pkg/adapters/sqlstore.Audit()` now correctly scans `created_at` from both PostgreSQL (time.Time) and SQLite (string) drivers

### Coverage

All packages brought to ≥85%: `pkg/migrate` (85%), `internal/cli` (74%), `internal/introspect` (95%), `pkg/tenant` (100%), `pkg/pagination` (100%), `pkg/server` (37% unit + integration).

---

## [0.4.0] — 2026-06-02

### Added

- Stable `gomyadmin init` starter with `.gitignore`, backend migrations, demo seed data, and a richer Next.js admin UI.
- Public `server.AdminStore` adapter interface for custom databases, ORMs, service layers, and cache-backed deployments.
- `server.Config.SessionStore` for Redis, Memcached, SQL, or existing session/cache integrations.
- `pkg/cache` with a minimal cache interface and in-memory implementation for adapter tests and local development.
- Bulk delete API at `POST /admin/api/{resource}/bulk-delete` with de-duplicated ids and audit logging.
- `pkg/adapters/sqlstore` for `database/sql` integrations, including MySQL and SQLite-style drivers.
- `pkg/adapters/gormstore`, `pkg/adapters/mongostore`, and `pkg/adapters/redisstore`.
- `pkg/migrate` SQL migration runner with ordered files and checksum tracking.
- Compatibility policy and examples for existing chi, Gin/Echo, and GORM apps.
- `docs/cli-init.md` for the supported init flags and generated project layout.
- `docs/drop-in-adapters.md` for mounting GoMyAdmin on existing Go backends.
- `docs/migrations.md` for versioned schema changes and release note expectations.
- Expanded `docs/auth.md` with session, CSRF, authorization, and production guidance.
- `examples/postgres-crm` with a real PostgreSQL schema and seed data for introspection.

### Changed

- Default generated module path now uses `github.com/darwvin-dev/<project>` when `--module` is omitted.
- Generated frontend projects now include a dashboard, resources page, shared API helper, and production-oriented Next.js config.

---

## [0.3.0] — 2026-05-24

### Added

**Drop-in HTTP handler (`pkg/server`)**
- `New(ctx, Config) (*AdminServer, error)` — connects to PostgreSQL (or accepts an existing pool), runs `CREATE TABLE IF NOT EXISTS` for `gomyadmin_sessions`, `gomyadmin_audit_logs`, and `gomyadmin_files`, then returns a ready server
- `AdminServer.Handler() http.Handler` — mounts all admin API routes under `/admin/api/`; compatible with any Go HTTP multiplexer (`net/http`, chi, gorilla/mux, echo, …)
- `AdminServer.Close()` — only closes the pool when `DatabaseURL` was supplied; callers who pass their own `Pool` retain ownership
- `Config.Authenticate` callback — plug in any existing user authentication logic; nil → 401 on all logins
- `Config.Tenants` callback — optional tenant list for the "me" endpoint
- `Config.Uploads` field — override the default local file storage with S3/R2/MinIO
- `camelToSnake` — converts `*admin.Field` names (CamelCase) to SQL column names (`"CreatedAt"→"created_at"`, `"TenantID"→"tenant_id"`, `"APIKey"→"api_key"`)
- Auto-generates primary keys: UUID v4 for `FieldUUID` primary keys; random hex otherwise
- Hashes `FieldPassword` values with Argon2id on create and update
- Full CRUD, bulk actions, CSV export, file upload/download, and audit log endpoints
- CORS, rate-limited login, session cookie, CSRF token — all wired in automatically

---

## [0.2.0] — 2026-05-23

### Added

**S3-compatible storage adapter (`pkg/storage`)**
- `NewS3(S3Config) (*S3Store, error)` — zero-dependency S3/R2/MinIO adapter using AWS Signature Version 4
- Implements `Storage` (Put, Get, Delete, SignedURL) and `Inspector` (Stat)
- Virtual-hosted and path-style URL modes; `ForcePathStyle` for R2 and MinIO
- `PublicBaseURL` populates the `StoredObject.URL` on Stat
- Presigned GET URLs generated entirely client-side (no round-trip)
- Parses S3-compatible XML error responses into descriptive Go errors

**Schema-driven resource generation (`internal/generator`, `internal/cli`)**
- `gomyadmin generate from-schema <schema.json>` — reads the JSON produced by `gomyadmin introspect` and writes one resource file per table in a single pass
- Column names are converted to CamelCase Go identifiers with acronym expansion (`id`→`ID`, `api`→`API`, `url`→`URL`)
- Table names are singularized and CamelCased (`users`→`User`, `api_keys`→`APIKey`, `categories`→`Category`)
- Field attributes are derived automatically: email/password heuristics, searchable, sortable, filterable, readonly on timestamps and primary keys
- `GeneratedField` gains `Primary` and `Readonly` flags; the resource template emits `.Primary()` and `.Readonly()` when set
- Default ID field in `GenerateResource` now sets `Primary: true, Readonly: true`

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

[0.5.0]: https://github.com/darwvin-dev/gomyadmin/releases/tag/v0.5.0
[0.4.0]: https://github.com/darwvin-dev/gomyadmin/releases/tag/v0.4.0
[0.3.0]: https://github.com/darwvin-dev/gomyadmin/releases/tag/v0.3.0
[0.2.0]: https://github.com/darwvin-dev/gomyadmin/releases/tag/v0.2.0
[0.1.0]: https://github.com/darwvin-dev/gomyadmin/releases/tag/v0.1.0
