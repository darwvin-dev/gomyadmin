package mongostore

import (
	"testing"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
	"go.mongodb.org/mongo-driver/bson"
)

// helpers to build test ResourceMeta without a real MongoDB connection.

type item struct{}

func testResource() server.ResourceMeta {
	app := admin.New("Test")
	app.Resource(item{}).
		TableName("items").
		TenantScoped("tenant_id").
		Field("ID").String().Primary().
		Field("Name").String().Searchable().Filterable().Sortable()

	meta := server.NewResourceMetadataStore(app)
	resources := meta.Resources()
	if len(resources) == 0 {
		panic("no resources registered")
	}
	return resources[0]
}

func TestBuildFilterEmpty(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "", "super_admin", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filter) != 0 {
		t.Fatalf("expected empty filter, got %v", filter)
	}
}

func TestBuildFilterTenantScoping(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "tenant-1", "viewer", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if filter["tenant_id"] != "tenant-1" {
		t.Fatalf("tenant_id = %v", filter["tenant_id"])
	}
}

func TestBuildFilterSuperAdminSkipsTenantScope(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "tenant-1", "super_admin", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := filter["tenant_id"]; ok {
		t.Fatal("super_admin should not be tenant-scoped")
	}
}

func TestBuildFilterSearch(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "", "super_admin", "alice", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := filter["$or"]; !ok {
		t.Fatal("expected $or for search")
	}
}

func TestBuildFilterEqOperator(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::eq": "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	if filter["name"] != "Bob" {
		t.Fatalf("name filter = %v", filter["name"])
	}
}

func TestBuildFilterContainsOperator(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::contains": "ob"})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := filter["name"].(bson.M)
	if !ok || m["$regex"] == nil {
		t.Fatalf("expected regex filter, got %v", filter["name"])
	}
}

func TestBuildFilterStartsWithOperator(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::starts_with": "Al"})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := filter["name"].(bson.M)
	if !ok {
		t.Fatalf("expected bson.M, got %T", filter["name"])
	}
	regex := m["$regex"].(string)
	if len(regex) == 0 || regex[0] != '^' {
		t.Fatalf("expected anchored regex, got %q", regex)
	}
}

func TestBuildFilterEndsWithOperator(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::ends_with": "ce"})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := filter["name"].(bson.M)
	if !ok {
		t.Fatalf("expected bson.M, got %T", filter["name"])
	}
	regex := m["$regex"].(string)
	if len(regex) == 0 || regex[len(regex)-1] != '$' {
		t.Fatalf("expected dollar-anchored regex, got %q", regex)
	}
}

func TestBuildFilterGteOperator(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::gte": "M"})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := filter["name"].(bson.M)
	if !ok || m["$gte"] == nil {
		t.Fatalf("expected $gte filter, got %v", filter["name"])
	}
}

func TestBuildFilterUnknownFieldReturnsError(t *testing.T) {
	resource := testResource()
	_, err := buildFilter(resource, "", "super_admin", "", map[string]string{"nonexistent::eq": "x"})
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestBuildFilterUnsupportedOperatorReturnsError(t *testing.T) {
	resource := testResource()
	_, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::badop": "x"})
	if err == nil {
		t.Fatal("expected error for unsupported operator")
	}
}

func TestBuildFilterEmptyValueSkipped(t *testing.T) {
	resource := testResource()
	filter, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::eq": ""})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := filter["name"]; ok {
		t.Fatal("empty filter value should be skipped")
	}
}

func TestBuildSortAscending(t *testing.T) {
	resource := testResource()
	sort := buildSort(resource, "name")
	if sort == nil {
		t.Fatal("expected sort, got nil")
	}
	if sort[0].Key != "name" || sort[0].Value != 1 {
		t.Fatalf("sort = %v", sort)
	}
}

func TestBuildSortDescending(t *testing.T) {
	resource := testResource()
	sort := buildSort(resource, "-name")
	if sort == nil {
		t.Fatal("expected sort, got nil")
	}
	if sort[0].Key != "name" || sort[0].Value != -1 {
		t.Fatalf("sort = %v", sort)
	}
}

func TestBuildSortUnknownFieldReturnsNil(t *testing.T) {
	resource := testResource()
	sort := buildSort(resource, "unknown_field")
	if sort != nil {
		t.Fatalf("expected nil sort for unknown field, got %v", sort)
	}
}

func TestNormalizeRecordRenamesUnderscoreID(t *testing.T) {
	record := server.Record{"_id": "mongo-obj-id", "name": "Alice"}
	normalized := normalizeRecord(record)
	if normalized["id"] != "mongo-obj-id" {
		t.Fatalf("id = %v", normalized["id"])
	}
	if _, ok := normalized["_id"]; ok {
		t.Fatal("_id should be removed after normalization")
	}
}

func TestNormalizeRecordPreservesExistingID(t *testing.T) {
	record := server.Record{"_id": "mongo-obj-id", "id": "existing-id", "name": "Bob"}
	normalized := normalizeRecord(record)
	if normalized["id"] != "existing-id" {
		t.Fatalf("existing id should not be overwritten, got %v", normalized["id"])
	}
}

func TestFieldBySQLName(t *testing.T) {
	resource := testResource()
	field, ok := fieldBySQLName(resource, "name")
	if !ok {
		t.Fatal("expected to find 'name' field")
	}
	if field.SQLName != "name" {
		t.Fatalf("sql name = %q", field.SQLName)
	}
}

func TestFieldBySQLNameMissing(t *testing.T) {
	resource := testResource()
	_, ok := fieldBySQLName(resource, "nonexistent")
	if ok {
		t.Fatal("should not find nonexistent field")
	}
}
