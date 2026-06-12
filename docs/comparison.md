# Comparison

GoMyAdmin is built for Go teams that want a generated admin panel they can own in their repository.

## When GoMyAdmin Fits

Use GoMyAdmin when:

- your backend is written in Go
- your main operational database is PostgreSQL
- you want generated Go and TypeScript files you can commit
- your admin UI needs custom actions, audit logs, RBAC, tenant scoping, and file uploads
- you prefer explicit code over runtime configuration magic
- you want to mount the admin panel inside an existing Go service

Avoid GoMyAdmin when:

- you need a hosted no-code product
- your team does not want to maintain generated code
- your primary data source is not PostgreSQL
- you need BI dashboards, charts, and reporting more than CRUD workflows
- non-engineers need to build workflows without touching code

## Alternatives

| Tool | Best For | Tradeoff |
|---|---|---|
| Retool | Fast internal tools with hosted/managed workflows | External platform dependency and less code ownership |
| Forest Admin | Managed admin over existing data models | Hosted service and vendor-specific workflow |
| AdminJS | Node.js admin panels | JavaScript/Node runtime, not Go-native |
| Django Admin | Django projects | Excellent inside Django, not useful for Go services |
| Prisma Studio | Inspecting Prisma-managed data | Not a production admin workflow |
| Hand-written admin panel | Maximum control | Slow to build and easy to under-test |
| GoMyAdmin | Go + PostgreSQL admin panels with committable generated code | Younger project, narrower PostgreSQL-first scope |

## Positioning

GoMyAdmin is not a low-code platform. It is closer to a code generator and integration kit:

```text
schema -> generated resources -> Go admin API -> Next.js admin UI
```

The generated app should be treated like application code. Review it, edit it, test it, and deploy it with the rest of your system.

## Why Not Just Build It By Hand?

Many admin panels repeat the same foundations:

- authentication and sessions
- resource metadata
- searchable tables
- create/update forms
- filters and sorting
- audit log
- RBAC
- file uploads
- tenant scoping
- CSV export

GoMyAdmin handles those repetitive parts while keeping the generated code visible.

## Why PostgreSQL First?

PostgreSQL gives the project a concrete target for introspection, SQL generation, migrations, filtering, and test coverage. A narrow database target keeps the first production path easier to reason about.

Other storage paths should be added only where they can be supported without weakening the PostgreSQL-first experience.
