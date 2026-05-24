# Stable CLI Init Flow

`gomyadmin init <name>` creates a complete starter app with a Go backend, PostgreSQL database, and Next.js admin UI.

```sh
gomyadmin init acme-admin --backend go --db postgres --frontend next --ui shadcn
cd acme-admin
cp .env.example .env
docker compose up --build
```

## What Gets Created

```text
backend/
  cmd/server/main.go
  internal/admin/resources.go
  internal/db/migrations/001_init.sql
  internal/db/seeds/001_demo.sql
frontend/
  app/admin/dashboard/page.tsx
  app/admin/login/page.tsx
  app/admin/resources/page.tsx
docker-compose.yml
Makefile
.env.example
```

## Supported Flags

```text
--backend go
--db postgres
--frontend next
--ui shadcn
--module github.com/you/acme-admin
```

Only the documented stack is supported in v0.1.x. Unsupported values fail fast so generated projects stay predictable.

## Recommended First Edits

1. Replace `GOMYADMIN_SESSION_SECRET` in `.env`.
2. Replace demo credentials and password hashes.
3. Edit `backend/internal/admin/resources.go`.
4. Add new migrations instead of editing `001_init.sql` after a release.
5. Run `make test` before committing generated changes.
