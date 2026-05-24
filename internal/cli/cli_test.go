package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
