# Free Demo Deploy

This guide prepares a public GoMyAdmin demo using free tiers:

- Render Free web services for the Go API and Next.js UI
- Neon Free Postgres for the demo database

The free setup is good for a public demo, but it is not meant for production traffic. Render free services can sleep after inactivity, so the first request after a quiet period may be slow.

## Architecture

```text
Browser
  -> gomyadmin-demo-web.onrender.com       Next.js admin UI
  -> gomyadmin-demo-api.onrender.com       Go admin API
  -> Neon Postgres                         demo data
```

## 1. Create Neon Postgres

Create a free Neon project and copy the pooled Postgres connection string.

Use it as:

```text
DATABASE_URL=postgresql://USER:PASSWORD@HOST/DB?sslmode=require
```

The demo backend creates its tables and seed data at startup.

## 2. Create the Render Blueprint

In Render:

1. New -> Blueprint
2. Connect `darwvin-dev/gomyadmin`
3. Select the root `render.yaml`
4. Fill the required environment variables

For `gomyadmin-demo-api`:

```text
DATABASE_URL=<your Neon pooled connection string>
GOMYADMIN_PUBLIC_URL=https://gomyadmin-demo-web.onrender.com
```

Render generates `GOMYADMIN_SESSION_SECRET`.

For `gomyadmin-demo-web`:

```text
NEXT_PUBLIC_ADMIN_API_URL=https://gomyadmin-demo-api.onrender.com
```

The frontend value is intentionally marked `sync: false` because `NEXT_PUBLIC_*` values are embedded during the Next.js build.

## 3. Deploy Order

Deploy the API first so you know the API URL, then set `NEXT_PUBLIC_ADMIN_API_URL` on the web service and redeploy the frontend.

After both services are live, set `GOMYADMIN_PUBLIC_URL` on the API to the web service URL and redeploy the API.

## 4. Test

Open:

```text
https://gomyadmin-demo-web.onrender.com/admin
```

Login:

```text
admin@example.com / password
```

Check:

- login succeeds
- resource list loads
- customers or invoices render
- audit log loads
- API CORS works from the frontend URL

## 5. Add the Live Link

After the demo is live, update the README:

```md
Live demo: https://gomyadmin-demo-web.onrender.com/admin

Login: `admin@example.com` / `password`
```

## Notes

- Free Render services may sleep.
- Keep the demo database separate from any real data.
- Rotate `GOMYADMIN_SESSION_SECRET` if the service config is shared.
- Neon storage and compute limits are enough for a small public demo, but watch usage after launch posts.
