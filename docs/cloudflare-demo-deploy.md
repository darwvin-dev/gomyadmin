# Cloudflare Demo Deploy

This guide creates a public GoMyAdmin demo without adding a payment card.

The Cloudflare setup is intentionally demo-only:

- Cloudflare Worker: lightweight public demo API
- Cloudflare Workers + OpenNext: existing Next.js admin UI
- No database secret required
- No changes to the production Go backend path

Use [free-demo-deploy.md](free-demo-deploy.md) if you want the Render + Neon path that runs the real Go backend with Postgres.

## Architecture

```text
Browser
  -> gomyadmin-demo.<account>.workers.dev      Next.js admin UI
  -> gomyadmin-demo-api.<account>.workers.dev  demo API Worker
```

The Worker mirrors the API contract expected by the frontend and returns seeded CRM data. It is not meant to replace the Go backend.

## 1. Deploy The Worker API

Install Wrangler on demand and deploy from the Worker directory:

```sh
cd deploy/cloudflare/worker
npm install
npx wrangler login
npx wrangler deploy
```

After deploy, copy the Worker URL. It will look like:

```text
https://gomyadmin-demo-api.<your-account>.workers.dev
```

Test the API:

```sh
curl https://gomyadmin-demo-api.<your-account>.workers.dev/admin/api/resources
```

## 2. Deploy The Frontend To Cloudflare Workers

Cloudflare recommends the Workers/OpenNext path for full Next.js apps. Deploy the existing frontend template from its directory:

```sh
cd templates/frontend-next-shadcn
yarn install --frozen-lockfile
NEXT_PUBLIC_ADMIN_API_URL=https://gomyadmin-demo-api.<your-account>.workers.dev npx wrangler deploy
```

Wrangler can auto-detect Next.js and generate the OpenNext Worker configuration. If Cloudflare asks for a project name, use:

```text
gomyadmin-demo
```

Open the deployed frontend:

```text
https://gomyadmin-demo.<your-account>.workers.dev/admin
```

Login:

```text
admin@example.com / password
```

## 3. Lock CORS To The Frontend URL

After the frontend is live, edit `deploy/cloudflare/worker/wrangler.toml`:

```toml
[vars]
ADMIN_ORIGIN = "https://gomyadmin-demo.<your-account>.workers.dev"
```

Redeploy the Worker:

```sh
cd deploy/cloudflare/worker
npx wrangler deploy
```

## 4. Test The Demo

Check:

- login succeeds
- resource list loads
- customers filter and search work
- export downloads CSV
- audit log loads
- creating, editing, and deleting records returns success responses

The Worker uses seeded demo data, so writes are response-level demos and do not persist between requests.

## 5. Add The Live Link

After the demo is live, update the README:

```md
Live demo: https://gomyadmin-demo.<your-account>.workers.dev/admin

Login: `admin@example.com` / `password`
```

## Notes

- Do not put Neon or production database URLs in public frontend environment variables.
- This Worker does not need `DATABASE_URL`.
- If you previously shared a Neon connection string, rotate the Neon password before using that database for anything public.
- For a production-like demo with real Postgres persistence, use the Render + Neon path or add a separate persistent Worker backend later.
