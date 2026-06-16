# Cloudflare Demo Deploy

A live demo built with this guide is running at:

```text
Frontend: https://gomyadmin-next-shadcn.darwvin-dev.workers.dev/admin/login
API:      https://gomyadmin-demo-api.darwvin-dev.workers.dev
Login:    admin@example.com / password
```

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
  -> gomyadmin-next-shadcn.<account>.workers.dev  Next.js admin UI (OpenNext)
  -> gomyadmin-demo-api.<account>.workers.dev     demo API Worker
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

Cloudflare recommends the Workers/OpenNext path for full Next.js apps. The frontend template already ships the OpenNext config (`open-next.config.ts`, `wrangler.jsonc`, and the `deploy`/`preview` scripts), so no migration step is needed. Deploy from the template directory:

```sh
cd templates/frontend-next-shadcn
yarn install --frozen-lockfile
NEXT_PUBLIC_ADMIN_API_URL=https://gomyadmin-demo-api.<your-account>.workers.dev npm run deploy
```

The worker is named `gomyadmin-next-shadcn` in `wrangler.jsonc`. Open the deployed frontend:

```text
https://gomyadmin-next-shadcn.<your-account>.workers.dev/admin
```

### If the worker upload times out or fails with `fetch failed`

The final step uploads the worker bundle (~1 MB) as a single request. On some links (mobile tethering, restrictive networks, certain VPNs) that large upload is reset while small requests and asset uploads still succeed. Route Wrangler through a stable local proxy if you have one (for example an HTTP/SOCKS proxy on `127.0.0.1:10808`):

```sh
export HTTPS_PROXY=http://127.0.0.1:10808
export HTTP_PROXY=http://127.0.0.1:10808
NEXT_PUBLIC_ADMIN_API_URL=https://gomyadmin-demo-api.<your-account>.workers.dev npm run deploy
```

Wrangler honors these variables ("We'll use your proxy for fetch requests"). Otherwise retry from a more stable connection.

Login:

```text
admin@example.com / password
```

## 3. Lock CORS To The Frontend URL

After the frontend is live, edit `deploy/cloudflare/worker/wrangler.toml`:

```toml
[vars]
ADMIN_ORIGIN = "https://gomyadmin-next-shadcn.<your-account>.workers.dev"
```

The demo API reflects the requesting origin in its CORS headers, so the demo works even before this step; locking `ADMIN_ORIGIN` is optional hardening.

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
Live demo: https://gomyadmin-next-shadcn.<your-account>.workers.dev/admin

Login: `admin@example.com` / `password`
```

## Notes

- Do not put Neon or production database URLs in public frontend environment variables.
- This Worker does not need `DATABASE_URL`.
- If you previously shared a Neon connection string, rotate the Neon password before using that database for anything public.
- For a production-like demo with real Postgres persistence, use the Render + Neon path or add a separate persistent Worker backend later.
