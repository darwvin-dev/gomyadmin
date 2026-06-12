package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

func TestUnknownCommandReturnsError(t *testing.T) {
	if code := Run([]string{"missing"}); code == 0 {
		t.Fatal("expected non-zero exit code")
	}
}

func TestGenerateFromSchemaMissingFile(t *testing.T) {
	if code := Run([]string{"generate", "from-schema", "nonexistent.json"}); code == 0 {
		t.Fatal("expected non-zero exit for missing schema file")
	}
}

func TestGenerateFromSchemaMissingArg(t *testing.T) {
	if code := Run([]string{"generate", "from-schema"}); code == 0 {
		t.Fatal("expected non-zero exit when schema path is omitted")
	}
}

func TestGenerateFromSchemaCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	schema := map[string]any{
		"tables": []map[string]any{
			{
				"name":   "users",
				"schema": "public",
				"columns": []map[string]any{
					{"name": "id", "data_type": "uuid", "nullable": false, "is_identity": false},
					{"name": "email", "data_type": "character varying", "nullable": false, "is_identity": false},
					{"name": "created_at", "data_type": "timestamp with time zone", "nullable": false, "is_identity": false},
				},
				"primary_key": []string{"id"},
			},
			{
				"name":   "invoices",
				"schema": "public",
				"columns": []map[string]any{
					{"name": "id", "data_type": "uuid", "nullable": false, "is_identity": false},
					{"name": "amount", "data_type": "numeric", "nullable": false, "is_identity": false},
				},
				"primary_key": []string{"id"},
			},
		},
	}

	schemaJSON, _ := json.Marshal(schema)
	schemaPath := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(schemaPath, schemaJSON, 0o644); err != nil {
		t.Fatal(err)
	}

	code := Run([]string{"generate", "from-schema", schemaPath, "--package", "adminapp"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	for _, tc := range []struct{ file, want string }{
		{"backend/internal/admin/user_resource.go", "type User struct"},
		{"backend/internal/admin/user_resource.go", `TableName("users")`},
		{"backend/internal/admin/user_resource.go", "Email()"},
		{"backend/internal/admin/invoice_resource.go", "type Invoice struct"},
		{"backend/internal/admin/invoice_resource.go", `TableName("invoices")`},
	} {
		data, err := os.ReadFile(filepath.Join(dir, tc.file))
		if err != nil {
			t.Fatalf("file %s not found: %v", tc.file, err)
		}
		if !strings.Contains(string(data), tc.want) {
			t.Errorf("%s: missing %q", tc.file, tc.want)
		}
	}
}

func TestGenerateFromSchemaInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(schemaPath, []byte("not json"), 0o644)

	if code := Run([]string{"generate", "from-schema", schemaPath}); code == 0 {
		t.Fatal("expected non-zero exit for invalid JSON")
	}
}

func TestTableToResourceName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"users", "User"},
		{"invoices", "Invoice"},
		{"categories", "Category"},
		{"user_profiles", "UserProfile"},
		{"api_keys", "APIKey"},
	}
	for _, c := range cases {
		if got := tableToResourceName(c.in); got != c.want {
			t.Errorf("tableToResourceName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestColumnToFieldName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"id", "ID"},
		{"email", "Email"},
		{"created_at", "CreatedAt"},
		{"api_key", "APIKey"},
		{"user_id", "UserID"},
	}
	for _, c := range cases {
		if got := columnToFieldName(c.in); got != c.want {
			t.Errorf("columnToFieldName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRunNoArgsReturnsZero(t *testing.T) {
	if code := Run([]string{}); code != 0 {
		t.Fatalf("empty args: expected 0, got %d", code)
	}
}

func TestRunVersionPrintsAndReturnsZero(t *testing.T) {
	if code := Run([]string{"version"}); code != 0 {
		t.Fatalf("version: expected 0, got %d", code)
	}
}

func TestVersionMatchesReleaseTag(t *testing.T) {
	if Version != "1.0.2" {
		t.Fatalf("Version = %q, want 1.0.2", Version)
	}
}

func TestRunHelpReturnsZero(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		if code := Run([]string{arg}); code != 0 {
			t.Fatalf("help(%q): expected 0, got %d", arg, code)
		}
	}
}

func TestRunDoctorReturnsZeroOrOne(t *testing.T) {
	code := Run([]string{"doctor"})
	if code != 0 && code != 1 {
		t.Fatalf("doctor: expected 0 or 1, got %d", code)
	}
}

func TestRunMessageCommandsReturnZero(t *testing.T) {
	for _, cmd := range []string{"migrate", "seed"} {
		if code := Run([]string{cmd}); code != 0 {
			t.Fatalf("%s: expected 0, got %d", cmd, code)
		}
	}
}

func TestRunGenerateNoSubcommandReturnsTwo(t *testing.T) {
	if code := Run([]string{"generate"}); code != 2 {
		t.Fatalf("generate (no sub): expected 2, got %d", code)
	}
}

func TestRunGenerateResourceNoNameReturnsTwo(t *testing.T) {
	if code := Run([]string{"generate", "resource"}); code != 2 {
		t.Fatalf("generate resource (no name): expected 2, got %d", code)
	}
}

func TestRunGenerateResourceCreatesFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(orig) }()

	code := Run([]string{"generate", "resource", "Product", "--table", "products"})
	if code != 0 {
		t.Fatalf("generate resource: expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "backend/internal/admin/product_resource.go")); err != nil {
		t.Fatalf("resource file not created: %v", err)
	}
}

func TestRunInitNoNameReturnsTwo(t *testing.T) {
	if code := Run([]string{"init"}); code != 2 {
		t.Fatalf("init (no name): expected 2, got %d", code)
	}
}

func TestRunInitCreatesProject(t *testing.T) {
	dir := t.TempDir()
	projectName := filepath.Join(dir, "my-admin")
	code := Run([]string{"init", projectName})
	if code != 0 {
		t.Fatalf("init: expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(projectName, "docker-compose.yml")); err != nil {
		t.Fatalf("docker-compose.yml not created: %v", err)
	}
}

func TestRunDemoCreatesNamedProject(t *testing.T) {
	dir := t.TempDir()
	projectName := filepath.Join(dir, "launch-demo")
	code := Run([]string{"demo", projectName})
	if code != 0 {
		t.Fatalf("demo: expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(projectName, "docker-compose.yml")); err != nil {
		t.Fatalf("docker-compose.yml not created in requested demo directory: %v", err)
	}
}

func TestRunIntrospectNoURLReturnsTwo(t *testing.T) {
	orig := os.Getenv("DATABASE_URL")
	_ = os.Unsetenv("DATABASE_URL")
	defer func() {
		if orig != "" {
			_ = os.Setenv("DATABASE_URL", orig)
		}
	}()
	if code := Run([]string{"introspect"}); code != 2 {
		t.Fatalf("introspect (no URL): expected 2, got %d", code)
	}
}

func TestRunGenerateUnknownTargetReturnsTwo(t *testing.T) {
	if code := Run([]string{"generate", "unknown-target"}); code != 2 {
		t.Fatalf("generate unknown: expected 2, got %d", code)
	}
}

func TestRunOpenAPIGeneratesFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "openapi.json")
	code := Run([]string{"openapi", "generate", "--out", out})
	if code != 0 {
		t.Fatalf("openapi generate: expected 0, got %d", code)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("openapi.json not created: %v", err)
	}
}

func TestRunGenerateFrontendBackendAll(t *testing.T) {
	for _, target := range []string{"frontend", "backend", "all"} {
		if code := Run([]string{"generate", target}); code != 0 {
			t.Fatalf("generate %s: expected 0, got %d", target, code)
		}
	}
}

func TestRunGenerateActionPolicy(t *testing.T) {
	for _, target := range []string{"action", "policy"} {
		if code := Run([]string{"generate", target}); code != 0 {
			t.Fatalf("generate %s: expected 0, got %d", target, code)
		}
	}
}

func TestSingularize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"users", "user"},
		{"categories", "category"},
		{"boxes", "box"},
		{"classes", "class"},
		{"buses", "bus"},
		{"passes", "pass"},
		{"person", "person"},
		{"ss", "ss"},
	}
	for _, c := range cases {
		if got := singularize(c.in); got != c.want {
			t.Errorf("singularize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFieldTypeMethodName(t *testing.T) {
	cases := []struct {
		ft   admin.FieldType
		want string
	}{
		{admin.FieldDateTime, "DateTime"},
		{admin.FieldUUID, "UUID"},
		{admin.FieldJSON, "JSON"},
		{admin.FieldJSONB, "JSONB"},
		{admin.FieldMoney, "Decimal"},
		{admin.FieldString, "String"},
		{admin.FieldEmail, "Email"},
		{admin.FieldBoolean, "Boolean"},
		{admin.FieldInteger, "Integer"},
	}
	for _, c := range cases {
		if got := fieldTypeMethodName(c.ft); got != c.want {
			t.Errorf("fieldTypeMethodName(%q) = %q, want %q", c.ft, got, c.want)
		}
	}
}
