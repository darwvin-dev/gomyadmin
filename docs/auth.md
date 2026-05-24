# Authentication

The auth package includes Argon2id password hashing, HTTP-only sessions, CSRF helpers, logout, current-user patterns, and rate limiting. Generated applications should use secure cookies in production and rotate the demo credentials immediately.

## Session Model

Generated apps use an HTTP-only session cookie created by `auth.SessionManager`. The session value points to server-side session state, so user roles, tenant IDs, and permissions can be checked without exposing them to browser JavaScript.

Default development behavior:

- Login endpoint: `POST /admin/api/auth/login`
- Logout endpoint: `POST /admin/api/auth/logout`
- Current user endpoint: `GET /admin/api/me`
- Session cookie: HTTP-only, same-site, path-scoped to the app
- Demo credentials: `admin@example.com` / `password`

Production requirements:

- Set a strong `GOMYADMIN_SESSION_SECRET`.
- Serve over HTTPS and enable secure cookies.
- Replace the demo password hash before exposing the app.
- Store sessions in PostgreSQL or Redis when running more than one backend instance.
- Keep session TTLs short for admin surfaces.

Existing applications can pass any implementation of `auth.SessionStore` through `server.Config.SessionStore`. This is the integration point for Redis, Memcached, SQL-backed sessions, or an existing internal session service.

## CSRF

Login issues a CSRF token cookie. Mutating admin requests should send the token back in `X-CSRF-Token`. The generated API already allows this header in CORS.

Recommended frontend pattern:

```ts
await fetch("/admin/api/users", {
  method: "POST",
  credentials: "include",
  headers: {
    "Content-Type": "application/json",
    "X-CSRF-Token": csrfToken
  },
  body: JSON.stringify(payload)
})
```

## Authorization

Every request should resolve an `admin.Actor` from session context. Resource permissions use strings such as:

```text
users.view
users.create
users.update
users.delete
users.actions.block_user
audit.view
files.create
```

Use `TenantScoped("tenant_id")` on resources that must be isolated per tenant. The backend template passes the actor tenant into list, get, create, update, and delete operations.

## Rate Limiting

The generated login route is protected by `auth.NewRateLimiter`. Keep this limiter enabled in production and add IP-aware or account-aware storage if the app runs behind multiple replicas.

## Roadmap

OAuth providers, magic links, and TOTP modules are planned as optional auth adapters rather than mandatory framework features.
