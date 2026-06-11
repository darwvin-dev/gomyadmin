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

## API Keys

GoMyAdmin v0.6 adds optional API key authentication for machine-to-machine use cases. A valid key can be sent in either:

- `Authorization: Bearer <key>`
- `X-API-Key: <key>`

When the built-in PostgreSQL server adapter is active, `server.New` automatically provisions an API key store and exposes:

- `GET /admin/api/auth/api-keys`
- `POST /admin/api/auth/api-keys`
- `POST /admin/api/auth/api-keys/{id}/revoke`

Secrets are shown only once at creation time. Stored values are hashed, and each key tracks `expires_at`, `revoked_at`, and `last_used_at`.

## OAuth

OAuth remains optional and adapter-driven. Configure providers in `server.Config.OAuthProviders` and map external identities into local actors with `server.Config.ResolveOAuthActor`.

The built-in flow exposes:

- `GET /admin/api/auth/providers`
- `GET /admin/api/auth/oauth/{provider}/start`
- `GET /admin/api/auth/oauth/{provider}/callback`

GoMyAdmin signs the OAuth state cookie with `server.Config.SigningSecret` (defaults to `GOMYADMIN_SESSION_SECRET`).

Minimal example:

```go
srv, err := server.New(ctx, server.Config{
    DatabaseURL: os.Getenv("DATABASE_URL"),
    OAuthProviders: map[string]auth.OAuthProvider{
        "google": auth.GoogleOAuthProvider(
            os.Getenv("GOOGLE_CLIENT_ID"),
            os.Getenv("GOOGLE_CLIENT_SECRET"),
        ),
    },
    ResolveOAuthActor: func(ctx context.Context, provider string, identity auth.OAuthIdentity) (admin.Actor, bool, error) {
        if identity.Email == "" {
            return admin.Actor{}, false, nil
        }
        return admin.Actor{
            ID:          identity.Subject,
            Email:       identity.Email,
            Name:        identity.Name,
            Roles:       []string{"super_admin"},
            Permissions: []string{"*"},
        }, true, nil
    },
})
```

See [docs/oauth-google.md](oauth-google.md) for a concrete Google setup flow.

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

Magic links and TOTP remain planned as optional auth adapters rather than mandatory framework features.
