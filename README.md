# GoMyAdmin

Generate a commit-friendly admin panel for existing Go + PostgreSQL apps.

GoMyAdmin gives you a small CLI, a Go resource API, PostgreSQL introspection, and a Next.js backoffice template. The generated code is meant to be read, committed, customized, and deployed with your application.

[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/darwvin-dev/gomyadmin/actions/workflows/ci.yml/badge.svg)](https://github.com/darwvin-dev/gomyadmin/actions/workflows/ci.yml)

<p align="center">
  <img src="docs/demo-cli.svg" alt="GoMyAdmin CLI demo" width="720" /><br/>
  <img src="docs/demo-ui.svg" alt="GoMyAdmin admin UI screenshot" width="720" />
</p>

## Why

Most Go teams eventually need an internal admin panel for users, billing, support, content, operations, or CRM data.

You can hand-write it, wire a low-code tool into production, or adopt a framework that hides too much. GoMyAdmin takes a narrower path:

```text
PostgreSQL schema -> resource definitions in Go -> generated admin API + Next.js UI
```

The output is explicit Go, SQL, and TypeScript. There is no runtime model magic and no requirement to replace your router, ORM, auth, or deployment flow.

## Quick Start

```sh
go install github.com/darwvin-dev/gomyadmin/cmd/gomyadmin@latest

gomyadmin demo my-admin
cd my-admin
cp .env.example .env
docker compose up --build
```

Open `http://localhost:3000/admin` and use:

```text
admin@example.com / password
```

Public demo deployment instructions are in [docs/free-demo-deploy.md](docs/free-demo-deploy.md). Once a hosted demo is live, add its URL here.

You can also start from an existing database:

```sh
gomyadmin introspect --database-url "$DATABASE_URL" > schema.json
gomyadmin generate from-schema schema.json
```

## What You Get

- A CLI for scaffolding, schema introspection, resource generation, OpenAPI output, and local checks.
- A Go resource builder for fields, actions, permissions, audit logging, and tenant scoping.
- A PostgreSQL-backed admin API with search, sort, filters, pagination, CRUD, bulk actions, file uploads, sessions, CSRF, RBAC, and audit logs.
- A Next.js + TypeScript + Tailwind + shadcn/ui admin template.
- Generated files that stay in your repo and can be edited by hand.

## Resource Example

```go
import "github.com/darwvin-dev/gomyadmin/pkg/admin"

app := admin.New("Acme Admin")

app.Resource(User{}).
    Label("Users").
    TableName("users").
    Field("ID").UUID().Primary().Readonly().
    Field("Email").Email().Required().Searchable().Sortable().Unique().
    Field("Name").String().Searchable().
    Field("Role").Enum("admin", "manager", "support", "viewer").Filterable().
    Field("Status").Enum("active", "blocked", "pending").Filterable().Badge().
    Field("CreatedAt").DateTime().Readonly().Sortable().DateRangeFilter().
    Action("Block User", blockUserHandler).Danger().RequireConfirmation().
    Audit().
    TenantScoped("tenant_id")
```

## CLI

```sh
gomyadmin init <name> [--backend go] [--db postgres] [--frontend next] [--ui shadcn]
gomyadmin demo [name]
gomyadmin introspect --database-url "$DATABASE_URL"
gomyadmin generate resource <Name> [--table <table>] [--package <pkg>] [--force]
gomyadmin generate from-schema <schema.json> [--package <pkg>] [--force]
gomyadmin doctor
gomyadmin openapi generate [--out openapi.json]
gomyadmin version
```

## Install As A Library

```sh
go get github.com/darwvin-dev/gomyadmin@latest
```

Mount the admin handler on your existing Go server:

```go
srv, err := server.New(ctx, server.Config{
    DatabaseURL:  os.Getenv("DATABASE_URL"),
    App:          app,
    Authenticate: verifyAdminCredentials,
})
if err != nil {
    log.Fatal(err)
}
defer srv.Close()

mux.Handle("/admin/", srv.Handler())
```

## Scope

GoMyAdmin is PostgreSQL-first and optimized for Go applications that want generated, committable code.

It is not trying to be a hosted low-code platform, a BI tool, or a universal replacement for every admin framework. Optional adapters exist for integration points such as sessions, storage, cache, ORM-backed access, and document resources, but the main path is Go + PostgreSQL.

See [comparison notes](docs/comparison.md) for when GoMyAdmin is a good fit.

## Documentation

- [Getting started](docs/getting-started.md)
- [Installation](docs/installation.md)
- [Resource definitions](docs/resource-definition.md)
- [Fields](docs/fields.md)
- [Actions](docs/actions.md)
- [Authentication and sessions](docs/auth.md)
- [RBAC](docs/rbac.md)
- [Audit log](docs/audit-log.md)
- [Multi-tenancy](docs/multi-tenancy.md)
- [Drop-in adapters](docs/drop-in-adapters.md)
- [Deployment](docs/deployment.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Comparison](docs/comparison.md)
- [Free demo deploy](docs/free-demo-deploy.md)

## Verification

```sh
go test ./...
go vet ./...

cd templates/frontend-next-shadcn
yarn install --frozen-lockfile
yarn run typecheck
yarn run build
```

## Roadmap

- [x] CLI install and runnable local demo
- [x] PostgreSQL introspection
- [x] Resource generation from schema JSON
- [x] Go resource builder API
- [x] Admin API with CRUD, filters, search, sort, pagination, actions, audit, RBAC, sessions, and files
- [x] Next.js admin template
- [x] Drop-in HTTP handler for existing Go backends
- [ ] Hosted public demo
- [ ] Relation field rendering in the frontend
- [ ] Playwright e2e coverage for the CRM demo
- [ ] Split heavier optional adapters into separate modules or documented opt-in packages

## Contributing

Small focused PRs are welcome. Good first areas:

- docs and examples
- relation-field UI
- demo polish
- adapter docs
- e2e coverage

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)
