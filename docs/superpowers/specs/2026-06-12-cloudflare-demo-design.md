# Cloudflare Demo Design

## Goal

Ship a card-free public GoMyAdmin demo on Cloudflare while keeping the main Go package and production backend unchanged.

## Architecture

The demo uses Cloudflare Workers/OpenNext for the existing Next.js admin UI and a small Cloudflare Worker for the public demo API. The Worker mirrors the admin API contract expected by `templates/frontend-next-shadcn/lib/api.ts`, returns seeded sample CRM data, and supports login, metadata, list/detail, create/update/delete response flows, exports, audit, and CORS. It is intentionally demo-only and does not replace the Go backend.

## Scope

In scope:

- Add a demo Worker under `deploy/cloudflare/worker`.
- Add Node-based contract tests for Worker route behavior.
- Add Cloudflare deployment docs.
- Link Cloudflare docs from README and launch docs.
- Keep the Render/Docker deployment path intact.

Out of scope:

- Rewriting the Go backend for Workers.
- Adding persistent storage to the demo Worker.
- Storing Neon credentials in the repository.
- Changing frontend UI behavior beyond deployment documentation.

## API Behavior

The Worker returns the same envelope shape used by the Go backend:

```json
{ "data": {}, "meta": {}, "error": null }
```

Errors use:

```json
{ "data": null, "meta": {}, "error": { "code": "INVALID_CREDENTIALS", "message": "Invalid email or password" } }
```

The public demo accepts:

```text
admin@example.com / password
```

All routes include CORS headers suitable for a Cloudflare-hosted frontend calling a Worker API from another Cloudflare subdomain.

## Deployment

The user deploys the API Worker with Wrangler, then deploys the existing frontend template from `templates/frontend-next-shadcn` with Wrangler's Next.js/OpenNext flow. The frontend build receives `NEXT_PUBLIC_ADMIN_API_URL` pointing at the API Worker.

## Testing

Worker behavior is covered by Node's built-in test runner, without pulling runtime dependencies into the root package. The normal Go verification remains unchanged.
