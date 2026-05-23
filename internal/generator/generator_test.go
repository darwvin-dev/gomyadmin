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
		".env.example",
		"docker-compose.yml",
		"Makefile",
		filepath.Join("backend", "Dockerfile"),
		filepath.Join("backend", "go.mod"),
		filepath.Join("backend", "cmd", "server", "main.go"),
		filepath.Join("backend", "internal", "admin", "resources.go"),
		filepath.Join("frontend", "Dockerfile"),
		filepath.Join("frontend", "package.json"),
	}
	for _, rel := range mustExist {
		path := filepath.Join(name, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", rel, err)
		}
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
