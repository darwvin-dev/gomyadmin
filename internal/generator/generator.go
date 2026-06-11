package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type InitOptions struct {
	Name     string
	Module   string
	Backend  string
	Database string
	Frontend string
	UI       string
}

func InitProject(options InitOptions) error {
	if strings.TrimSpace(options.Name) == "" {
		return errors.New("project name is required")
	}
	if options.Module == "" {
		options.Module = "github.com/darwvin-dev/" + moduleName(options.Name)
	}
	if options.Backend == "" {
		options.Backend = "go"
	}
	if options.Database == "" {
		options.Database = "postgres"
	}
	if options.Frontend == "" {
		options.Frontend = "next"
	}
	if options.UI == "" {
		options.UI = "shadcn"
	}
	if options.Backend != "go" {
		return fmt.Errorf("unsupported backend %q", options.Backend)
	}
	if options.Database != "postgres" {
		return fmt.Errorf("unsupported database %q", options.Database)
	}
	if options.Frontend != "next" {
		return fmt.Errorf("unsupported frontend %q", options.Frontend)
	}
	if options.UI != "shadcn" {
		return fmt.Errorf("unsupported ui %q", options.UI)
	}

	root := filepath.Clean(options.Name)
	if root == "." || root == string(filepath.Separator) {
		return errors.New("project name must be a new directory, not the current directory")
	}
	files, err := projectFiles(options)
	if err != nil {
		return err
	}
	if err := ensureNoConflicts(root, files); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	written := make([]string, 0, len(files))
	defer func() {
		if err == nil {
			return
		}
		for i := len(written) - 1; i >= 0; i-- {
			_ = os.Remove(written[i])
		}
		_ = os.Remove(root)
	}()
	for name, content := range files {
		path := filepath.Join(root, name)
		if writeErr := writeFile(path, content, false); writeErr != nil {
			err = writeErr
			return err
		}
		written = append(written, path)
	}
	return nil
}

func projectFiles(options InitOptions) (map[string]string, error) {
	files := map[string]string{}
	for path, templateBody := range map[string]string{
		"README.md":                           readmeTemplate,
		".gitignore":                          gitignoreTemplate,
		".env.example":                        envTemplate,
		"docker-compose.yml":                  composeTemplate,
		"Makefile":                            makefileTemplate,
		"backend/Dockerfile":                  backendDockerfileTemplate,
		"backend/go.mod":                      backendGoModTemplate,
		"backend/cmd/server/main.go":          backendMainTemplate,
		"backend/internal/admin/resources.go": resourcesTemplate,
		"backend/internal/db/migrations/001_init.sql": migrationTemplate,
		"backend/internal/db/seeds/001_demo.sql":      seedTemplate,
		"frontend/Dockerfile":                         frontendDockerfileTemplate,
		"frontend/package.json":                       frontendPackageTemplate,
		"frontend/next.config.js":                     frontendNextConfigTemplate,
		"frontend/tsconfig.json":                      frontendTSConfigTemplate,
		"frontend/next-env.d.ts":                      frontendNextEnvTemplate,
		"frontend/app/layout.tsx":                     frontendLayoutTemplate,
		"frontend/app/globals.css":                    frontendGlobalsTemplate,
		"frontend/app/admin/page.tsx":                 frontendAdminRedirectTemplate,
		"frontend/app/admin/dashboard/page.tsx":       frontendDashboardTemplate,
		"frontend/app/admin/login/page.tsx":           loginPageTemplate,
		"frontend/app/admin/resources/page.tsx":       frontendResourcesTemplate,
		"frontend/lib/api.ts":                         frontendAPITemplate,
	} {
		rendered, renderErr := render(templateBody, options)
		if renderErr != nil {
			return nil, renderErr
		}
		files[path] = rendered
	}
	return files, nil
}

type ResourceOptions struct {
	Name      string
	Package   string
	Table     string
	Fields    []GeneratedField
	Overwrite bool
}

type GeneratedField struct {
	Name       string
	Type       string
	Primary    bool
	Readonly   bool
	Searchable bool
	Sortable   bool
	Filterable bool
	Required   bool
}

func GenerateResource(options ResourceOptions) error {
	if options.Name == "" {
		return errors.New("resource name is required")
	}
	if options.Package == "" {
		options.Package = "adminapp"
	}
	if options.Table == "" {
		options.Table = pluralSnake(options.Name)
	}
	if len(options.Fields) == 0 {
		options.Fields = []GeneratedField{
			{Name: "ID", Type: "UUID", Primary: true, Readonly: true, Sortable: true},
			{Name: "Name", Type: "String", Searchable: true, Sortable: true, Required: true},
			{Name: "CreatedAt", Type: "DateTime", Readonly: true, Sortable: true, Filterable: true},
		}
	}
	if err := os.MkdirAll(filepath.Join("backend", "internal", "admin"), 0o755); err != nil {
		return err
	}

	content, err := render(resourceTemplate, options)
	if err != nil {
		return err
	}
	formatted, err := format.Source([]byte(content))
	if err == nil {
		content = string(formatted)
	}
	path := filepath.Join("backend", "internal", "admin", resourceFileName(options.Name))
	return writeFile(path, content, options.Overwrite)
}

func writeFile(path, content string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; pass --force to overwrite", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func render(body string, data any) (string, error) {
	t, err := template.New("template").Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"slug":  moduleName,
	}).Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ensureNoConflicts(root string, files map[string]string) error {
	for name := range files {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; remove it or choose another project directory", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func resourceFileName(name string) string {
	return strings.ReplaceAll(strings.ToLower(splitWords(name)), " ", "_") + "_resource.go"
}

func moduleName(name string) string {
	base := filepath.Base(filepath.Clean(name))
	base = strings.TrimSpace(strings.ToLower(base))
	var b strings.Builder
	lastDash := false
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "gomyadmin-app"
	}
	return slug
}

func pluralSnake(name string) string {
	base := strings.ReplaceAll(strings.ToLower(splitWords(name)), " ", "_")
	if strings.HasSuffix(base, "s") || strings.HasSuffix(base, "x") || strings.HasSuffix(base, "z") || strings.HasSuffix(base, "ch") || strings.HasSuffix(base, "sh") {
		return base + "es"
	}
	if strings.HasSuffix(base, "y") && len(base) > 1 {
		prev := base[len(base)-2]
		if !strings.ContainsRune("aeiou", rune(prev)) {
			return base[:len(base)-1] + "ies"
		}
	}
	return base + "s"
}

func splitWords(name string) string {
	var words []string
	var current []rune
	runes := []rune(strings.TrimSpace(name))
	flush := func() {
		if len(current) == 0 {
			return
		}
		words = append(words, string(current))
		current = nil
	}
	for i, r := range runes {
		if r == '_' || r == '-' || r == ' ' {
			flush()
			continue
		}
		if len(current) > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || ((prev >= 'A' && prev <= 'Z') && nextLower && len(current) > 1) {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()
	return strings.Join(words, " ")
}

const readmeTemplate = `# {{.Name}}

Generated with GoMyAdmin.

## Run locally

` + "```sh" + `
cp .env.example .env
docker compose up --build
` + "```" + `

Open http://localhost:3000/admin and login with admin@example.com / password.

## Generated stack

- Go backend in ./backend
- PostgreSQL schema in ./backend/internal/db/migrations
- Demo seed data in ./backend/internal/db/seeds
- Next.js admin UI in ./frontend
- Session cookies and CSRF-ready auth endpoints

## Common commands

` + "```sh" + `
make dev      # run the full stack
make backend  # run only the Go backend
make frontend # run only the Next.js frontend
make test     # run backend tests
` + "```" + `
`

const gitignoreTemplate = `.env
.DS_Store
node_modules/
.next/
dist/
tmp/
coverage/
.gocache/
*.log
`

const envTemplate = `DATABASE_URL=postgres://gomyadmin:gomyadmin@postgres:5432/gomyadmin?sslmode=disable
GOMYADMIN_SESSION_SECRET=change-me-before-production
GOMYADMIN_PUBLIC_URL=http://localhost:3000
GOMYADMIN_BACKEND_URL=http://localhost:8080
NEXT_PUBLIC_ADMIN_API_URL=http://localhost:8080
`

const composeTemplate = `services:
  postgres:
    image: postgres:17
    environment:
      POSTGRES_USER: gomyadmin
      POSTGRES_PASSWORD: gomyadmin
      POSTGRES_DB: gomyadmin
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U gomyadmin"]
      interval: 5s
      timeout: 5s
      retries: 20

  backend:
    build: ./backend
    env_file: .env
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy

  frontend:
    build: ./frontend
    environment:
      NEXT_PUBLIC_ADMIN_API_URL: http://localhost:8080
    ports:
      - "3000:3000"
    depends_on:
      - backend
`

const makefileTemplate = `.PHONY: demo dev backend frontend test migrate seed

demo:
	docker compose up --build

dev:
	docker compose up --build

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

test:
	cd backend && go test ./...

migrate:
	docker compose exec -T postgres psql -U gomyadmin -d gomyadmin -f /dev/stdin < backend/internal/db/migrations/001_init.sql

seed:
	docker compose exec -T postgres psql -U gomyadmin -d gomyadmin -f /dev/stdin < backend/internal/db/seeds/001_demo.sql
`

const backendDockerfileTemplate = `FROM golang:1.25 AS build
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o /out/server ./cmd/server

FROM debian:bookworm-slim
COPY --from=build /out/server /server
EXPOSE 8080
CMD ["/server"]
`

const backendGoModTemplate = `module {{.Module}}/backend

go 1.25.0

require github.com/darwvin-dev/gomyadmin v0.6.0
`

const backendMainTemplate = `package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	adminapp "{{.Module}}/backend/internal/admin"
)

func main() {
	handler := adminapp.Handler()
	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Println("admin backend listening on :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
`

const frontendDockerfileTemplate = `FROM node:22-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build
EXPOSE 3000
CMD ["npm", "run", "start"]
`

const resourcesTemplate = `package admin

import (
	"net/http"

	forge "github.com/darwvin-dev/gomyadmin/pkg/admin"
)

type User struct{}
type Organization struct{}
type Invoice struct{}
type Role struct{}

func Handler() http.Handler {
	app := forge.New("{{.Name}}")
	RegisterResources(app)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forge.WriteJSON(w, http.StatusOK, r.Header.Get("X-Request-ID"), map[string]any{
			"name": app.Name(),
			"resources": app.Resources(),
		}, nil)
	})
}

func RegisterResources(app *forge.App) {
	app.Resource(User{}).
		Label("Users").
		Icon("users").
		Description("Manage application users").
		Field("ID").UUID().Primary().Readonly().
		Field("Email").Email().Required().Searchable().Sortable().Unique().
		Field("Name").String().Searchable().Sortable().
		Field("Role").Enum("admin", "manager", "support", "viewer").Filterable().
		Field("Status").Enum("active", "blocked", "pending").Filterable().Badge().
		Field("TenantID").TenantKey().Hidden().
		Field("CreatedAt").DateTime().Readonly().DateRangeFilter().
		Action("Block User", nil).Danger().RequireConfirmation().RequireReason().
		Action("Reset Password", nil).RequireConfirmation().
		Audit().
		TenantScoped("tenant_id")

	app.Resource(Organization{}).Label("Organizations").Icon("building").Audit()
	app.Resource(Invoice{}).Label("Invoices").Icon("receipt").Audit().TenantScoped("tenant_id")
	app.Resource(Role{}).Label("Roles").Icon("shield").Audit()
}
`

const migrationTemplate = `create table if not exists tenants (
  id text primary key,
  name text not null,
  slug text not null unique,
  created_at timestamptz not null default now()
);

create table if not exists users (
  id text primary key,
  tenant_id text not null references tenants(id),
  email text not null unique,
  name text not null,
  role text not null default 'admin',
  status text not null default 'active',
  password_hash text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists customers (
  id text primary key,
  tenant_id text not null references tenants(id),
  name text not null,
  email text not null,
  status text not null default 'lead',
  plan text not null default 'starter',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists invoices (
  id text primary key,
  tenant_id text not null references tenants(id),
  customer_id text not null references customers(id) on delete cascade,
  number text not null,
  amount numeric(12,2) not null default 0,
  status text not null default 'open',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists users_tenant_created_idx on users (tenant_id, created_at desc);
create index if not exists customers_tenant_created_idx on customers (tenant_id, created_at desc);
create index if not exists invoices_tenant_status_idx on invoices (tenant_id, status, created_at desc);
`

const seedTemplate = `insert into tenants (id, name, slug)
values ('tenant_demo', 'Northstar CRM', 'northstar')
on conflict (id) do nothing;

insert into users (id, tenant_id, email, name, role, status, password_hash)
values ('user_admin', 'tenant_demo', 'admin@example.com', 'Darwin Admin', 'admin', 'active', '')
on conflict (id) do nothing;

insert into customers (id, tenant_id, name, email, status, plan)
values
  ('cust_acme', 'tenant_demo', 'Acme Telecom', 'ops@acme.test', 'active', 'scale'),
  ('cust_nova', 'tenant_demo', 'Nova Logistics', 'admin@nova.test', 'lead', 'starter')
on conflict (id) do nothing;

insert into invoices (id, tenant_id, customer_id, number, amount, status)
values
  ('inv_1001', 'tenant_demo', 'cust_acme', 'INV-1001', 4800.00, 'open'),
  ('inv_1002', 'tenant_demo', 'cust_nova', 'INV-1002', 900.00, 'paid')
on conflict (id) do nothing;
`

const frontendPackageTemplate = `{
  "name": "{{slug .Name}}-frontend",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "lucide-react": "^0.511.0",
    "next": "15.3.2",
    "react": "19.1.0",
    "react-dom": "19.1.0"
  },
  "devDependencies": {
    "@types/node": "22.15.30",
    "@types/react": "19.1.6",
    "@types/react-dom": "19.1.5",
    "typescript": "5.8.3"
  }
}
`

const frontendNextConfigTemplate = `/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone"
}

module.exports = nextConfig
`

const frontendTSConfigTemplate = `{
  "compilerOptions": {
    "target": "ES2017",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": false,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./*"]
    }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx"],
  "exclude": ["node_modules"]
}
`

const frontendNextEnvTemplate = `/// <reference types="next" />
/// <reference types="next/image-types/global" />
`

const frontendLayoutTemplate = `import type { Metadata } from "next"
import type { ReactNode } from "react"
import "./globals.css"

export const metadata: Metadata = {
  title: "{{.Name}} Admin",
  description: "Generated GoMyAdmin backoffice"
}

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
`

const frontendGlobalsTemplate = `:root {
  color-scheme: light;
  --background: #f7f8fb;
  --panel: #ffffff;
  --foreground: #172033;
  --muted: #eef2f6;
  --border: #d8dee8;
  --brand: #126b5f;
  --brand-strong: #0b4d44;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  background: var(--background);
  color: var(--foreground);
  font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

a {
  color: inherit;
  text-decoration: none;
}

button,
input,
select {
  font: inherit;
}

.shell {
  display: grid;
  min-height: 100vh;
  grid-template-columns: 260px 1fr;
}

.sidebar {
  border-right: 1px solid var(--border);
  background: var(--panel);
  padding: 20px 14px;
}

.brand {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 8px;
  font-weight: 700;
}

.brand-mark {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 8px;
  background: var(--brand);
  color: white;
}

.nav {
  display: grid;
  gap: 4px;
  margin-top: 22px;
}

.nav a {
  display: flex;
  min-height: 38px;
  align-items: center;
  gap: 10px;
  border-radius: 8px;
  padding: 8px 10px;
  color: #536176;
}

.nav a:hover,
.nav a:first-child {
  background: var(--muted);
  color: var(--foreground);
}

.content {
  min-width: 0;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  min-height: 58px;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border);
  background: rgba(247, 248, 251, 0.92);
  padding: 0 24px;
  backdrop-filter: blur(12px);
}

.page {
  padding: 24px;
}

.page-title {
  margin: 0;
  font-size: 28px;
  line-height: 1.2;
}

.subtle {
  color: #66758a;
}

.grid {
  display: grid;
  gap: 16px;
}

.stats {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-top: 22px;
}

.card {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--panel);
  padding: 16px;
  box-shadow: 0 1px 2px rgba(19, 32, 51, 0.06);
}

.mt {
  margin-top: 16px;
}

.card-title {
  margin: 0 0 8px;
  color: #66758a;
  font-size: 14px;
}

.metric {
  font-size: 28px;
  font-weight: 700;
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th,
.table td {
  border-bottom: 1px solid var(--border);
  padding: 12px;
  text-align: left;
}

.button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 0;
  border-radius: 8px;
  background: var(--brand);
  color: white;
  padding: 0 14px;
  font-weight: 600;
}

.login {
  display: grid;
  min-height: 100vh;
  place-items: center;
  padding: 24px;
}

.login form {
  display: grid;
  width: min(100%, 380px);
  gap: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--panel);
  padding: 24px;
}

.input {
  min-height: 40px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0 12px;
}

@media (max-width: 900px) {
  .shell {
    grid-template-columns: 1fr;
  }

  .sidebar {
    position: static;
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .topbar,
  .page {
    padding-inline: 16px;
  }

  .stats {
    grid-template-columns: 1fr;
  }
}
`

const frontendAdminRedirectTemplate = `import { redirect } from "next/navigation"

export default function AdminPage() {
  redirect("/admin/dashboard")
}
`

const frontendDashboardTemplate = `import Link from "next/link"
import { ArrowUpRight, Database, FileText, KeyRound, ShieldCheck, Users } from "lucide-react"
import { apiURL } from "@/lib/api"

const stats = [
  { label: "Resources", value: "4", icon: Database },
  { label: "Admin users", value: "1", icon: Users },
  { label: "Open invoices", value: "1", icon: FileText },
  { label: "API auth", value: "v0.6", icon: KeyRound },
  { label: "Session policy", value: "Secure", icon: ShieldCheck }
]

export default function DashboardPage() {
  return (
    <main className="shell">
      <aside className="sidebar">
        <Link className="brand" href="/admin/dashboard">
          <span className="brand-mark">G</span>
          <span>{{.Name}}</span>
        </Link>
        <nav className="nav">
          <Link href="/admin/dashboard"><Database size={16} /> Dashboard</Link>
          <Link href="/admin/resources"><FileText size={16} /> Resources</Link>
          <Link href="/admin/login"><Users size={16} /> Login</Link>
        </nav>
      </aside>
      <section className="content">
        <header className="topbar">
          <span className="subtle">Backend: {apiURL}</span>
          <Link className="button" href="/admin/resources">Resources <ArrowUpRight size={16} /></Link>
        </header>
        <div className="page">
          <h1 className="page-title">Operations dashboard</h1>
          <p className="subtle">A focused starter for managing PostgreSQL-backed resources with Go.</p>
          <section className="grid stats">
            {stats.map((stat) => {
              const Icon = stat.icon
              return (
                <article className="card" key={stat.label}>
                  <p className="card-title">{stat.label}</p>
                  <div className="metric">{stat.value}</div>
                  <Icon size={18} color="#126b5f" />
                </article>
              )
            })}
          </section>
          <section className="card mt">
            <h2>Ready for real data</h2>
            <p className="subtle">Run migrations and seeds, then wire OAuth providers and API keys into your deployment config.</p>
          </section>
        </div>
      </section>
    </main>
  )
}
`

const loginPageTemplate = `export default function LoginPage() {
  return (
    <main className="login">
      <form>
        <h1>Sign in</h1>
        <p className="subtle">Use admin@example.com / password after replacing the demo password hash. OAuth providers can be mounted at /admin/api/auth/oauth/:provider/start.</p>
        <input className="input" placeholder="admin@example.com" />
        <input className="input" placeholder="password" type="password" />
        <button className="button">Continue</button>
      </form>
    </main>
  )
}
`

const frontendResourcesTemplate = `import Link from "next/link"

const resources = [
  { name: "users", label: "Users", description: "Admin users, roles, and account status" },
  { name: "customers", label: "Customers", description: "Tenant-scoped customer records" },
  { name: "invoices", label: "Invoices", description: "Billing records and payment state" },
  { name: "roles", label: "Roles", description: "Access control profiles" }
]

export default function ResourcesPage() {
  return (
    <main className="page">
      <h1 className="page-title">Resources</h1>
      <p className="subtle">Generated from the starter PostgreSQL schema.</p>
      <section className="card mt">
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            {resources.map((resource) => (
              <tr key={resource.name}>
                <td><Link href={"/admin/resources/" + resource.name}>{resource.label}</Link></td>
                <td className="subtle">{resource.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </main>
  )
}
`

const frontendAPITemplate = `export const apiURL = process.env.NEXT_PUBLIC_ADMIN_API_URL ?? "http://localhost:8080"

export async function adminRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(apiURL + path, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    },
    ...init
  })
  if (!response.ok) {
    throw new Error("Admin API request failed")
  }
  return response.json() as Promise<T>
}

export function oauthStartURL(provider: string): string {
  return apiURL + "/admin/api/auth/oauth/" + provider + "/start"
}
`

const resourceTemplate = `package {{.Package}}

import "github.com/darwvin-dev/gomyadmin/pkg/admin"

type {{.Name}} struct{}

func Register{{.Name}}Resource(app *admin.App) {
	app.Resource({{.Name}}{}).
		Label("{{.Name}}").
		TableName("{{.Table}}").
{{- range .Fields }}
		Field("{{.Name}}").{{.Type}}(){{ if .Primary }}.Primary(){{ end }}{{ if .Readonly }}.Readonly(){{ end }}{{ if .Required }}.Required(){{ end }}{{ if .Searchable }}.Searchable(){{ end }}{{ if .Sortable }}.Sortable(){{ end }}{{ if .Filterable }}.Filterable(){{ end }}.
{{- end }}
		Audit()
}
`
