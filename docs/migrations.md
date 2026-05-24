# Versioned Migrations

GoMyAdmin generated apps keep database changes in `backend/internal/db/migrations`.

## Naming

Use monotonically increasing filenames:

```text
001_init.sql
002_add_customer_notes.sql
003_add_invoice_due_date.sql
```

Each migration should be safe to run once in every environment. Prefer explicit `create table`, `alter table`, and `create index` statements. Use `if not exists` for early project templates where developers may rerun local setup.

## Upgrade Flow

1. Create a new migration file.
2. Add or update seed data only in `backend/internal/db/seeds`.
3. Run the migration locally.
4. Run `go test ./...`.
5. Commit the migration with the code that depends on it.

## Release Notes

For package releases, document schema-impacting changes in `CHANGELOG.md`:

```text
## v0.2.0
- Added invoice due dates.
- Migration: add `invoices.due_date date`.
```

Generated apps own their production migration runner. The starter Makefile is intentionally simple and suitable for local development.
