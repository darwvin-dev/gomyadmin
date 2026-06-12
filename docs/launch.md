# Launch Checklist

This checklist is for preparing GoMyAdmin for public distribution and repeatable installs.

## Product Readiness

- Keep the README short and focused on the first successful install.
- Keep the first path centered on Go + PostgreSQL.
- Avoid claiming broad production maturity without examples, tests, or user reports.
- Maintain a working `go install github.com/darwvin-dev/gomyadmin/cmd/gomyadmin@latest` path.
- Keep `gomyadmin demo my-admin` runnable without extra flags.

## Demo

A hosted demo should show the product before asking users to install anything.

Use [cloudflare-demo-deploy.md](cloudflare-demo-deploy.md) for the card-free Cloudflare path. Use [free-demo-deploy.md](free-demo-deploy.md) for the Render + Neon path that runs the real Go backend.

Recommended demo:

```text
https://demo.gomyadmin.dev
admin@example.com / password
```

The demo should include:

- users
- customers
- invoices
- payments
- audit log
- file uploads
- one custom action
- one tenant-scoped resource

Reset demo data on a schedule so visitors can safely edit records.

## Release Flow

For each release:

```sh
go test ./...
go vet ./...

cd templates/frontend-next-shadcn
yarn install --frozen-lockfile
yarn run typecheck
yarn run build
```

Then:

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```

Create a GitHub release with:

- install command
- short feature summary
- breaking changes
- upgrade notes
- known limitations

## Distribution Channels

After the hosted demo and README are ready:

- submit a short post to `r/golang`
- post a concise Show HN
- publish a Go Forum announcement
- open a PR to a relevant awesome-go list only after the project has docs, tests, and a demo
- write one technical article showing how to build an admin panel from an existing PostgreSQL schema

## Suggested Launch Copy

```text
I built GoMyAdmin: a Go + PostgreSQL admin panel generator that creates readable Go and TypeScript code you can commit, customize, and deploy.

It can introspect an existing PostgreSQL schema, generate admin resources, and run a local Next.js backoffice with a Go API.
```

## Good First Issues

Keep a small set of contributor-friendly issues open:

- add screenshots from the hosted demo
- add a Supabase/Postgres example
- add a chi middleware integration guide
- add relation-field frontend rendering
- add Playwright e2e coverage for the CRM demo
- document how to run GoMyAdmin behind a reverse proxy
- split heavier optional adapters into separate modules or opt-in packages

## Metrics To Watch

- stars
- clones
- `go install` issues
- docs page visits if a docs site exists
- issue quality
- external mentions
- accepted PRs from first-time contributors
