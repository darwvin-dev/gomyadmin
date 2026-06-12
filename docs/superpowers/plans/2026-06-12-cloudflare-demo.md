# Cloudflare Demo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a card-free Cloudflare live demo path for GoMyAdmin.

**Architecture:** Keep the Go backend as the production reference. Add a separate Cloudflare Worker demo API with seeded data and contract tests, then document how to deploy the API Worker and the existing Next.js frontend template to Cloudflare Workers.

**Tech Stack:** Cloudflare Workers, OpenNext for Cloudflare, JavaScript modules, Node `node:test`, Next.js frontend template.

---

### Task 1: Worker Contract Tests

**Files:**
- Create: `deploy/cloudflare/worker/test/demo-api.test.mjs`

- [ ] **Step 1: Write failing tests**

Create tests that import `createDemoWorker()` from `../src/demo-api.mjs`, call `worker.fetch()`, and assert:

- `OPTIONS /admin/api/resources` returns CORS preflight headers.
- `POST /admin/api/auth/login` accepts `admin@example.com` and `password`.
- `GET /admin/api/resources` returns a non-empty resource metadata list.
- `GET /admin/api/customers?page=1&per_page=2&q=acme` returns matching rows and pagination metadata.
- `GET /admin/api/customers/export` returns CSV.
- `POST`, `PATCH`, and `DELETE` resource routes return success envelopes.

- [ ] **Step 2: Run tests to verify red**

Run:

```bash
node --test deploy/cloudflare/worker/test/demo-api.test.mjs
```

Expected: FAIL because `deploy/cloudflare/worker/src/demo-api.mjs` does not exist.

### Task 2: Demo Worker Implementation

**Files:**
- Create: `deploy/cloudflare/worker/src/demo-api.mjs`
- Create: `deploy/cloudflare/worker/src/index.mjs`
- Create: `deploy/cloudflare/worker/package.json`
- Create: `deploy/cloudflare/worker/wrangler.toml`

- [ ] **Step 1: Implement minimal Worker**

Implement a dependency-free Worker module that exports `createDemoWorker()`. The Worker should route the admin API endpoints, return JSON envelopes, reflect CORS origins, and return seeded resources and rows.

- [ ] **Step 2: Run focused tests**

Run:

```bash
node --test deploy/cloudflare/worker/test/demo-api.test.mjs
```

Expected: PASS.

### Task 3: Cloudflare Deployment Docs

**Files:**
- Create: `docs/cloudflare-demo-deploy.md`
- Modify: `README.md`
- Modify: `docs/launch.md`

- [ ] **Step 1: Document deploy steps**

Write instructions for:

- Deploy Worker with `npx wrangler deploy`.
- Deploy `templates/frontend-next-shadcn` with the OpenNext Cloudflare migration and deploy script.
- Set `NEXT_PUBLIC_ADMIN_API_URL` to the Worker URL.
- Use `admin@example.com / password` to test the demo.
- Rotate or avoid database secrets because this Worker does not need Neon.

- [ ] **Step 2: Link docs**

Add Cloudflare demo links to the README and launch checklist.

### Task 4: Verification and Commit

**Files:**
- All files changed above.

- [ ] **Step 1: Run full verification**

Run:

```bash
node --test deploy/cloudflare/worker/test/demo-api.test.mjs
go test ./...
go vet ./...
python3 -c 'import yaml; yaml.safe_load(open("render.yaml"))'
```

Expected: all pass.

- [ ] **Step 2: Commit**

Run:

```bash
git add deploy/cloudflare docs README.md
git commit -m "deploy: add Cloudflare demo path"
```
