# Installation

```sh
go install github.com/darwvin/gomyadmin/cmd/gomyadmin@latest
```

Create an app:

```sh
gomyadmin init my-admin-app --backend go --db postgres --frontend next --ui shadcn
cd my-admin-app
docker compose up --build
```

GoMyAdmin targets Go `1.26`, PostgreSQL, and Next.js `16`.
