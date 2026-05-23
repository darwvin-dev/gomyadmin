# Contributing to GoMyAdmin

Thanks for helping make GoMyAdmin a serious Go backoffice framework.

## Local development

```sh
git clone https://github.com/darwvin/gomyadmin
cd gomyadmin
cp .env.example .env
make demo          # starts PostgreSQL + Go backend + Next.js frontend via Docker Compose
```

Run checks before opening a pull request:

```sh
go test ./...
go vet ./...
cd templates/frontend-next-shadcn && npm ci && npm run typecheck && npm run build
```

Or with Make:

```sh
make test          # go test ./... + frontend typecheck + build
make vet           # go vet ./...
```

## Where things live

| Path | What it does |
|---|---|
| `cmd/gomyadmin/` | CLI entry point |
| `internal/cli/` | Command routing and flag parsing |
| `internal/doctor/` | Environment prerequisite checks |
| `internal/generator/` | Project scaffold + resource file generation |
| `internal/introspect/` | PostgreSQL schema introspection + type mapping |
| `pkg/admin/` | Core resource metadata API |
| `pkg/auth/` | Passwords, sessions, CSRF, rate limiting |
| `pkg/audit/` | Audit event contracts and stores |
| `pkg/filters/` | Query parsing (search, sort, filter) |
| `pkg/openapi/` | OpenAPI 3.1 spec generation |
| `pkg/pagination/` | Page + per_page parsing |
| `pkg/postgres/` | Safe SQL query builder |
| `pkg/rbac/` | Role-based access control |
| `pkg/storage/` | File storage interface and adapters |
| `pkg/tenant/` | Multi-tenancy context and resolvers |
| `templates/` | Backend Go server + Next.js frontend templates |
| `examples/` | Domain-specific resource examples |
| `docs/` | Developer documentation |

## Design principles

- **Generated code is committed code.** Generated files should be readable, safe to edit by hand, and version-controlled alongside the application. Never generate files that cannot be opened in an editor and understood immediately.
- **Backend permissions are authoritative.** The frontend UI reflects what the backend allows. Access checks live in Go, not in TypeScript.
- **PostgreSQL first.** Introspection, filtering, sorting, and pagination all target PostgreSQL. Other databases are out of scope for now.
- **Security and tenancy changes need tests.** Every security boundary (CSRF, session, rate limit, RBAC, tenant isolation) must have tests that verify both the happy path and the failure mode.
- **Explicit over magic.** Prefer code that a reader can follow without knowing the framework internals.

## Pull request guidelines

- Open small PRs with a clear problem statement, implementation notes, and verification steps.
- For new packages or significant new features, include at minimum a unit test and a brief doc update.
- Security-sensitive changes must include failure-mode tests (e.g., token mismatch, expired session, blocked IP).
- Keep generated template code consistent with the documentation and examples.

## Reporting bugs

Open an issue with:
1. Go version (`go version`)
2. PostgreSQL version
3. What you did
4. What you expected
5. What happened (error output, stack trace)

## Reporting security issues

See [SECURITY.md](SECURITY.md). Do not file a public issue for vulnerabilities.
