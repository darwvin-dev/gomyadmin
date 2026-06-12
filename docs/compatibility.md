# Compatibility Policy

GoMyAdmin follows semantic versioning.

## Before v1.0.0

The project is still stabilizing. Minor releases may change public APIs when the change improves the long-term adapter model, security posture, or generated app structure.

Even before v1, GoMyAdmin tries to keep these stable:

- `pkg/admin` resource builder concepts
- `pkg/server.Config`
- `server.AdminStore`
- `auth.SessionStore`
- `storage.Storage`
- generated API route shapes under `/admin/api`

## v1.0.0 and Later

After v1.0.0:

- breaking public API changes require a major version
- adapters should remain source-compatible across minor releases
- generated apps may receive new files, but existing generated code should keep compiling
- database migrations are append-only

## Supported Runtime Targets

- Go: 1.25+
- HTTP: any stack that can mount `http.Handler`
- Built-in database adapter: PostgreSQL via pgx
- Optional adapters: database/sql, MySQL-style SQL, SQLite-style SQL, GORM, MongoDB
- Sessions: PostgreSQL, in-memory, cache-backed, Redis-backed
- Storage: local filesystem, in-memory, S3-compatible
