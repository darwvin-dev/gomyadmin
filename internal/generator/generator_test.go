package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateResourceCreatesFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	err := GenerateResource(ResourceOptions{
		Name:    "Customer",
		Package: "adminapp",
		Table:   "customers",
	})
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join("backend", "internal", "admin", "customer_resource.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("generated file not found: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "package adminapp") {
		t.Error("missing package declaration")
	}
	if !strings.Contains(content, "type Customer struct") {
		t.Error("missing type declaration")
	}
	if !strings.Contains(content, `TableName("customers")`) {
		t.Error("missing table name")
	}
	if !strings.Contains(content, "RegisterCustomerResource") {
		t.Error("missing register function")
	}
}

func TestGenerateResourceRejectsExistingFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	_ = GenerateResource(ResourceOptions{Name: "Order", Package: "adminapp"})

	err := GenerateResource(ResourceOptions{Name: "Order", Package: "adminapp"})
	if err == nil {
		t.Fatal("expected error for duplicate resource")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestGenerateResourceOverwriteFlag(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	_ = GenerateResource(ResourceOptions{Name: "Product", Package: "adminapp"})

	err := GenerateResource(ResourceOptions{Name: "Product", Package: "adminapp", Overwrite: true})
	if err != nil {
		t.Fatalf("overwrite should succeed: %v", err)
	}
}

func TestGenerateResourceDefaultsFields(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	err := GenerateResource(ResourceOptions{Name: "Tag"})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join("backend", "internal", "admin", "tag_resource.go"))
	content := string(data)
	if !strings.Contains(content, "ID") || !strings.Contains(content, "Name") {
		t.Error("default fields should include ID and Name")
	}
}

func TestInitProjectCreatesStructure(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "myapp")

	err := InitProject(InitOptions{
		Name:     name,
		Module:   "github.com/test/myapp",
		Backend:  "go",
		Database: "postgres",
		Frontend: "next",
		UI:       "shadcn",
	})
	if err != nil {
		t.Fatal(err)
	}

	mustExist := []string{
		"README.md",
		".gitignore",
		".env.example",
		"docker-compose.yml",
		"Makefile",
		filepath.Join("backend", "Dockerfile"),
		filepath.Join("backend", "go.mod"),
		filepath.Join("backend", "cmd", "server", "main.go"),
		filepath.Join("backend", "internal", "admin", "resources.go"),
		filepath.Join("backend", "internal", "db", "migrations", "001_init.sql"),
		filepath.Join("backend", "internal", "db", "seeds", "001_demo.sql"),
		filepath.Join("frontend", "Dockerfile"),
		filepath.Join("frontend", "package.json"),
		filepath.Join("frontend", "app", "layout.tsx"),
		filepath.Join("frontend", "app", "globals.css"),
		filepath.Join("frontend", "app", "admin", "dashboard", "page.tsx"),
		filepath.Join("frontend", "app", "admin", "resources", "page.tsx"),
		filepath.Join("frontend", "lib", "api.ts"),
	}
	for _, rel := range mustExist {
		path := filepath.Join(name, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", rel, err)
		}
	}
}

func TestInitProjectBackendUsesPortEnv(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "myapp")

	err := InitProject(InitOptions{
		Name:     name,
		Module:   "github.com/test/myapp",
		Backend:  "go",
		Database: "postgres",
		Frontend: "next",
		UI:       "shadcn",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(name, "backend", "cmd", "server", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `port := os.Getenv("PORT")`) {
		t.Fatalf("generated backend does not read PORT:\n%s", content)
	}
	if !strings.Contains(content, `Addr:              ":" + port,`) {
		t.Fatalf("generated backend does not listen on PORT:\n%s", content)
	}
}

func TestInitProjectDefaultsModuleToDarwvinDev(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "Acme Admin")

	if err := InitProject(InitOptions{Name: name}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(name, "backend", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "module github.com/darwvin-dev/acme-admin/backend") {
		t.Fatalf("unexpected module file:\n%s", string(data))
	}
}

func TestInitProjectRejectsUnsupportedBackend(t *testing.T) {
	err := InitProject(InitOptions{Name: "x", Backend: "rails", Database: "postgres", Frontend: "next", UI: "shadcn"})
	if err == nil || !strings.Contains(err.Error(), "unsupported backend") {
		t.Fatalf("error = %v", err)
	}
}

func TestInitProjectRejectsUnsupportedDatabase(t *testing.T) {
	err := InitProject(InitOptions{Name: "x", Backend: "go", Database: "mysql", Frontend: "next", UI: "shadcn"})
	if err == nil || !strings.Contains(err.Error(), "unsupported database") {
		t.Fatalf("error = %v", err)
	}
}

func TestInitProjectRejectsEmptyName(t *testing.T) {
	err := InitProject(InitOptions{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestInitProjectRejectsConflict(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "conflict-app")

	_ = InitProject(InitOptions{Name: name, Module: "github.com/test/conflict"})

	err := InitProject(InitOptions{Name: name, Module: "github.com/test/conflict"})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

// --- naming helpers ---

func TestPluralSnake(t *testing.T) {
	cases := []struct{ in, want string }{
		{"User", "users"},
		{"Invoice", "invoices"},
		{"Category", "categories"},
		{"Tax", "taxes"},
	}
	for _, c := range cases {
		if got := pluralSnake(c.in); got != c.want {
			t.Errorf("pluralSnake(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestResourceFileName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"User", "user_resource.go"},
		{"InvoiceItem", "invoice_item_resource.go"},
		{"APIKey", "api_key_resource.go"},
	}
	for _, c := range cases {
		if got := resourceFileName(c.in); got != c.want {
			t.Errorf("resourceFileName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
