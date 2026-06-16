# Supabase Postgres Example

This guide shows how to inspect a Supabase Postgres database, generate GoMyAdmin resources from the schema, and run the generated admin locally.

## 1. Prepare The Connection String

In the Supabase dashboard, open the project and click **Connect**. For most local development machines, copy the **Session pooler** connection string because it works on IPv4-only networks. Use the direct connection string when your network supports IPv6 or your project has the IPv4 add-on.

Set the connection string in your shell and replace the placeholders with your own project reference, region, and database password:

```sh
export DATABASE_URL='postgres://postgres.<project-ref>:<database-password>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require'
```

Keep this value in your local shell or an uncommitted `.env` file. Do not commit Supabase passwords or connection strings.

## 2. Introspect The Supabase Schema

Install the CLI and write the Supabase schema to `schema.json`:

```sh
go install github.com/darwvin-dev/gomyadmin/cmd/gomyadmin@latest
gomyadmin introspect --database-url "$DATABASE_URL" > schema.json
```

Inspect the file before generating resources:

```sh
head -40 schema.json
```

## 3. Generate A Local Admin App

Create a generated admin app and copy the schema into it:

```sh
gomyadmin init supabase-admin --backend go --db postgres --frontend next --ui shadcn
cp schema.json supabase-admin/schema.json
cd supabase-admin
```

Generate resource files from the Supabase schema:

```sh
gomyadmin generate from-schema schema.json --package admin --force
```

The generated files are written under `backend/internal/admin/`. Review them, then register the generated resource functions from `backend/internal/admin/resources.go`.

## 4. Run The Admin Locally

Copy the example environment and point the backend at Supabase:

```sh
cp .env.example .env
grep -v '^DATABASE_URL=' .env > .env.tmp
printf 'DATABASE_URL=%s\n' "$DATABASE_URL" > .env
cat .env.tmp >> .env
rm .env.tmp
```

Start the generated stack:

```sh
docker compose up --build
```

Open `http://localhost:3000/admin` and sign in with the generated app's local admin credentials:

```text
admin@example.com / password
```

If you run the backend without Docker, keep `DATABASE_URL` in your shell and start the services separately:

```sh
make backend
make frontend
```

## Notes

- Use SSL for hosted Supabase Postgres connections, so keep `sslmode=require` in the URL.
- Use a local `.env` file or secret manager for the connection string. Do not commit real credentials.
- The generated app's demo migrations and seed files are for local demo databases. When connecting to an existing Supabase project, inspect the live schema and generate resources from `schema.json` instead of running demo migrations against production data.
- Supabase's connection guide explains the direct connection, session pooler, and transaction pooler options: https://supabase.com/docs/guides/database/connecting-to-postgres
