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
		options.Module = "github.com/darwvin/" + filepath.Base(options.Name)
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

	root := options.Name
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
		".env.example":                        envTemplate,
		"docker-compose.yml":                  composeTemplate,
		"Makefile":                            makefileTemplate,
		"backend/Dockerfile":                  backendDockerfileTemplate,
		"backend/go.mod":                      backendGoModTemplate,
		"backend/cmd/server/main.go":          backendMainTemplate,
		"backend/internal/admin/resources.go": resourcesTemplate,
		"frontend/Dockerfile":                 frontendDockerfileTemplate,
		"frontend/package.json":               frontendPackageTemplate,
		"frontend/next.config.js":             frontendNextConfigTemplate,
		"frontend/tsconfig.json":              frontendTSConfigTemplate,
		"frontend/next-env.d.ts":              frontendNextEnvTemplate,
		"frontend/app/admin/page.tsx":         frontendPageTemplate,
		"frontend/app/admin/login/page.tsx":   loginPageTemplate,
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

Run:

` + "```sh" + `
cp .env.example .env
docker compose up
` + "```" + `

Open http://localhost:3000/admin and login with admin@example.com / password.

Notes:

- Generated stack: Go backend + PostgreSQL + Next.js frontend
- Supported flags today: --backend go --db postgres --frontend next --ui shadcn
- Pagination uses per_page before limit when both are present
`

const envTemplate = `DATABASE_URL=postgres://gomyadmin:gomyadmin@postgres:5432/gomyadmin?sslmode=disable
GOMYADMIN_SESSION_SECRET=change-me-before-production
GOMYADMIN_PUBLIC_URL=http://localhost:3000
GOMYADMIN_BACKEND_URL=http://localhost:8080
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

const makefileTemplate = `.PHONY: demo dev test

demo:
	docker compose up --build

dev:
	docker compose up postgres

test:
	cd backend && go test ./...
`

const backendDockerfileTemplate = `FROM golang:1.23 AS build
WORKDIR /app
COPY . .
RUN go build -o /out/server ./cmd/server

FROM debian:bookworm-slim
COPY --from=build /out/server /server
EXPOSE 8080
CMD ["/server"]
`

const backendGoModTemplate = `module {{.Module}}/backend

go 1.23

require github.com/darwvin-dev/gomyadmin v0.1.0
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

const frontendPackageTemplate = `{
  "name": "{{.Name}}-frontend",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
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
const nextConfig = {}

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
    "incremental": true
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx"],
  "exclude": ["node_modules"]
}
`

const frontendNextEnvTemplate = `/// <reference types="next" />
/// <reference types="next/image-types/global" />
`

const frontendPageTemplate = `export default function AdminPage() {
  return (
    <main className="p-8 font-sans">
      <h1>{{.Name}} Admin</h1>
      <p>Production-ready backoffice generated by GoMyAdmin.</p>
      <ul>
        <li>Backend URL: {process.env.NEXT_PUBLIC_ADMIN_API_URL ?? "http://localhost:8080"}</li>
        <li>Default page: /admin</li>
        <li>Login page: /admin/login</li>
      </ul>
    </main>
  )
}
`

const loginPageTemplate = `export default function LoginPage() {
  return (
    <main className="grid min-h-screen place-items-center font-sans">
      <form className="grid w-[360px] gap-3">
        <h1>Sign in</h1>
        <input placeholder="admin@example.com" />
        <input placeholder="password" type="password" />
        <button>Continue</button>
      </form>
    </main>
  )
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
