package openapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/darwvin/gomyadmin/pkg/admin"
)

func buildTestApp() *admin.App {
	app := admin.New("TestApp")
	type User struct{}
	type Invoice struct{}

	app.Resource(User{}).
		Label("Users").
		Field("ID").UUID().Primary().Readonly().
		Field("Email").Email().Required().Searchable().Sortable().
		Field("Role").Enum("admin", "viewer").Filterable().
		Field("Status").Enum("active", "blocked").Filterable().Badge().
		Action("Block User", nil).Danger().RequireReason()

	app.Resource(Invoice{}).
		Label("Invoices").
		Field("Amount").Money("USD").Sortable().
		Field("Status").Enum("paid", "open").Filterable().Badge().
		Action("Refund", nil).Danger()

	return app
}

func TestGenerateReturnsOpenAPI31(t *testing.T) {
	app := buildTestApp()
	spec := Generate(app, "0.1.0")
	if spec.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q", spec.OpenAPI)
	}
}

func TestGenerateIncludesAllResources(t *testing.T) {
	app := buildTestApp()
	spec := Generate(app, "0.1.0")

	if _, ok := spec.Paths["/admin/api/user"]; !ok {
		t.Error("missing /admin/api/user path")
	}
	if _, ok := spec.Paths["/admin/api/invoice"]; !ok {
		t.Error("missing /admin/api/invoice path")
	}
}

func TestGenerateIncludesListAndCreate(t *testing.T) {
	app := buildTestApp()
	spec := Generate(app, "0.1.0")

	userPath, ok := spec.Paths["/admin/api/user"]
	if !ok {
		t.Fatal("no user path")
	}
	if _, ok := userPath["get"]; !ok {
		t.Error("missing GET on user list path")
	}
	if _, ok := userPath["post"]; !ok {
		t.Error("missing POST on user list path")
	}
}

func TestGenerateIncludesDetailPatchDelete(t *testing.T) {
	app := buildTestApp()
	spec := Generate(app, "0.1.0")

	detail, ok := spec.Paths["/admin/api/user/{id}"]
	if !ok {
		t.Fatal("no user detail path")
	}
	for _, method := range []string{"get", "patch", "delete"} {
		if _, ok := detail[method]; !ok {
			t.Errorf("missing %s on user detail path", method)
		}
	}
}

func TestGenerateIncludesActionPaths(t *testing.T) {
	app := buildTestApp()
	spec := Generate(app, "0.1.0")

	actionPath := "/admin/api/user/{id}/actions/block-user"
	if _, ok := spec.Paths[actionPath]; !ok {
		t.Errorf("missing action path %s", actionPath)
	}
}

func TestGenerateIncludesBuiltinPaths(t *testing.T) {
	app := buildTestApp()
	spec := Generate(app, "0.1.0")

	for _, path := range []string{
		"/admin/api/resources",
		"/admin/api/me",
		"/admin/api/audit",
	} {
		if _, ok := spec.Paths[path]; !ok {
			t.Errorf("missing built-in path %s", path)
		}
	}
}

func TestJSONRoundtrip(t *testing.T) {
	app := buildTestApp()
	spec := Generate(app, "0.1.0")

	data, err := JSON(spec)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Spec
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON is not valid: %v", err)
	}
	if parsed.OpenAPI != "3.1.0" {
		t.Errorf("openapi after round-trip = %q", parsed.OpenAPI)
	}
}

func TestFieldSchemaTypes(t *testing.T) {
	app := admin.New("test")
	type X struct{}
	r := app.Resource(X{})
	r.Field("UUID").UUID()
	r.Field("Active").Boolean()
	r.Field("Count").Integer()
	r.Field("Price").Money("USD")
	r.Field("CreatedAt").DateTime()
	r.Field("Avatar").ImageUpload()
	r.Field("Status").Enum("a", "b").Badge()
	r.Field("Meta").JSONB()

	spec := Generate(app, "1.0.0")
	data, _ := JSON(spec)
	content := string(data)

	// json.MarshalIndent adds a space after the colon
	for _, want := range []string{
		`"format": "uuid"`,
		`"type": "boolean"`,
		`"type": "integer"`,
		`"type": "number"`,
		`"format": "date-time"`,
		`"format": "uri"`,
		`"type": "object"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("spec missing %q", want)
		}
	}
}

func TestInfoTitle(t *testing.T) {
	app := admin.New("Acme Admin")
	spec := Generate(app, "2.0.0")
	if spec.Info.Title != "Acme Admin Admin API" {
		t.Errorf("title = %q", spec.Info.Title)
	}
	if spec.Info.Version != "2.0.0" {
		t.Errorf("version = %q", spec.Info.Version)
	}
}
