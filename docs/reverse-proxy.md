# Reverse Proxy Deployment

GoMyAdmin can run behind Nginx, Caddy, Traefik, or a platform proxy as long as the public URL, forwarded headers, cookies, upload limits, and timeouts all agree.

The generated backend serves API routes under `/admin/api/`. The generated Next.js frontend serves the admin UI under `/admin`. In production, route both through HTTPS and keep the browser-facing origin stable.

## Public URL and Admin Path

Set the backend `PublicURL` to the externally reachable origin, including scheme and host:

```sh
GOMYADMIN_PUBLIC_URL=https://admin.example.com
```

`server.Config.PublicURL` defaults to `GOMYADMIN_PUBLIC_URL`, then `http://localhost:8080`. GoMyAdmin uses this value for:

- `Access-Control-Allow-Origin`
- OAuth redirect callback URLs such as `/admin/api/auth/oauth/google/callback`
- generated local file URLs such as `/admin/api/files/{id}`

Set the generated frontend API URL to the same public origin when the UI and API are served together. If they are split across origins, point it at the public backend origin instead. The development fallback is `http://localhost:8080`, so production deployments should set this explicitly:

```sh
NEXT_PUBLIC_ADMIN_API_URL=https://admin.example.com
```

Avoid publishing the admin UI under one path while sending API requests to another origin unless CORS and cookies are configured intentionally.

## Nginx Example

This example serves the Next.js admin frontend from `127.0.0.1:3000` and the Go backend from `127.0.0.1:8080` under one HTTPS origin.

```nginx
server {
    listen 443 ssl http2;
    server_name admin.example.com;

    client_max_body_size 25m;

    location /admin/api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 120s;
        proxy_send_timeout 120s;
    }

    location /admin/ {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 120s;
        proxy_send_timeout 120s;
    }
}
```

Keep the trailing slash on `location /admin/api/` so API routes continue to match GoMyAdmin's `/admin/api/...` handlers.

## Caddy Example

```caddyfile
admin.example.com {
    request_body {
        max_size 25MB
    }

    handle /admin/api/* {
        reverse_proxy 127.0.0.1:8080 {
            header_up Host {host}
            header_up X-Forwarded-Proto https
            header_up X-Forwarded-Host {host}
        }
    }

    handle /admin/* {
        reverse_proxy 127.0.0.1:3000 {
            header_up Host {host}
            header_up X-Forwarded-Proto https
            header_up X-Forwarded-Host {host}
        }
    }
}
```

Use `handle` instead of `handle_path` so Caddy does not strip the `/admin/api/` prefix before proxying.

## Cookies and CSRF

GoMyAdmin uses:

- `gomyadmin_session` for the HTTP-only admin session
- `gomyadmin_csrf` for the double-submit CSRF token
- `gomyadmin_oauth` for OAuth state when OAuth is enabled

Production deployments should:

- terminate TLS before requests reach users
- use one stable public origin for UI and API whenever possible
- keep `SameSite=Lax` behavior in mind when splitting UI and API across subdomains
- pass `X-CSRF-Token` on mutating admin API requests
- set `GOMYADMIN_SESSION_SECRET` to a strong private value

If login succeeds but later writes fail with `CSRF_FAILED`, confirm that the browser receives `gomyadmin_csrf`, sends it back in `X-CSRF-Token`, and that the proxy is not stripping cookies or custom headers.

## Upload Limits

File uploads go through `POST /admin/api/files`. Configure the proxy body limit to be at least as large as the maximum upload size you allow in the app or storage adapter.

Recommended checks:

- Nginx: set `client_max_body_size`.
- Caddy: set `request_body max_size`.
- Platform proxies: check request body and timeout limits.
- S3/R2/MinIO: use private buckets and signed URLs for sensitive files.

If uploads fail with `413 Request Entity Too Large`, raise the proxy limit before changing application code.

## Timeouts

Large exports, imports, slow database queries, and file uploads can outlive very short proxy defaults. Start with 120 seconds for `proxy_read_timeout` and `proxy_send_timeout`, then tighten based on production metrics.

Keep database statement timeouts and HTTP proxy timeouts aligned. A proxy timeout shorter than the backend timeout can make successful backend work look like an intermittent network failure to the browser.

## Troubleshooting Checklist

- `GOMYADMIN_PUBLIC_URL` exactly matches the browser-facing origin, including `https://`.
- `NEXT_PUBLIC_ADMIN_API_URL` points to the public backend URL instead of the localhost development fallback.
- `/admin/` reaches the Next.js frontend and `/admin/api/` reaches the Go backend.
- `Host`, `X-Forwarded-Host`, and `X-Forwarded-Proto` are forwarded.
- Login responses set `gomyadmin_session` and `gomyadmin_csrf`.
- Mutating requests include `X-CSRF-Token`.
- The proxy allows `Content-Type`, `X-CSRF-Token`, and `X-Request-ID` headers.
- Upload body limits are high enough for expected files.
- OAuth callback URLs registered with the provider match `/admin/api/auth/oauth/{provider}/callback`.
- Run `gomyadmin doctor` after changing deployment configuration.
