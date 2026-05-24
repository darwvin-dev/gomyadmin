package postgres

import (
	"net/url"
	"strings"
	"testing"

	"github.com/darwvin-dev/gomyadmin/pkg/filters"
)

func TestQueryBuilderUsesParameters(t *testing.T) {
	query, err := filters.Parse(url.Values{
		"q":            {"darwin"},
		"sort":         {"-created_at"},
		"filter[role]": {"admin"},
	}, map[string]filters.AllowedField{
		"email":      {Name: "email", Type: filters.TypeString, Searchable: true},
		"role":       {Name: "role", Type: filters.TypeEnum, Filterable: true},
		"created_at": {Name: "created_at", Type: filters.TypeDateTime, Sortable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	built, err := QueryBuilder{
		Table:        "users",
		TenantColumn: "tenant_id",
		Allowed: map[string]filters.AllowedField{
			"email":      {Name: "email", Type: filters.TypeString, Searchable: true},
			"role":       {Name: "role", Type: filters.TypeEnum, Filterable: true},
			"created_at": {Name: "created_at", Type: filters.TypeDateTime, Sortable: true},
		},
	}.Select(query, "tenant_1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(built.SQL, "darwin") || strings.Contains(built.SQL, "admin") {
		t.Fatalf("query contains raw values: %s", built.SQL)
	}
	if len(built.Args) != 5 {
		t.Fatalf("args = %#v", built.Args)
	}
}

func TestQueryBuilderAddsPrimaryKeyOrdering(t *testing.T) {
	query, err := filters.Parse(url.Values{}, map[string]filters.AllowedField{})
	if err != nil {
		t.Fatal(err)
	}
	built, err := QueryBuilder{
		Table:      "users",
		PrimaryKey: "id",
	}.Select(query, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(built.SQL, `order by "id" asc`) {
		t.Fatalf("sql = %s", built.SQL)
	}
}

func TestQueryBuilderCountValidatesTable(t *testing.T) {
	_, err := QueryBuilder{Table: `users;drop`}.Count(filters.Query{}, "")
	if err == nil {
		t.Fatal("expected invalid table error")
	}
}

func TestQueryBuilderRejectsUnknownSortField(t *testing.T) {
	query, err := filters.Parse(url.Values{"sort": {"created_at"}}, map[string]filters.AllowedField{
		"created_at": {Name: "created_at", Sortable: false},
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
	_ = query
}
