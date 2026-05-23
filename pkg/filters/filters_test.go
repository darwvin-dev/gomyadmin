package filters

import (
	"net/url"
	"testing"
)

func TestParseRejectsUnsortableField(t *testing.T) {
	_, err := Parse(url.Values{"sort": {"email"}}, map[string]AllowedField{
		"email": {Name: "email", Sortable: false},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAllowsWhitelistedFilters(t *testing.T) {
	query, err := Parse(url.Values{
		"q":                       {"darwin"},
		"sort":                    {"-created_at"},
		"filter[role]":            {"admin"},
		"filter[created_at][gte]": {"2026-01-01"},
	}, map[string]AllowedField{
		"created_at": {Name: "created_at", Type: TypeDateTime, Sortable: true, Filterable: true},
		"role":       {Name: "role", Type: TypeEnum, Filterable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if query.Search != "darwin" || len(query.Sorts) != 1 || len(query.Filters) != 2 {
		t.Fatalf("unexpected query: %#v", query)
	}
}

func TestParseSortsFiltersDeterministically(t *testing.T) {
	query, err := Parse(url.Values{
		"filter[b]": {"2"},
		"filter[a]": {"1"},
	}, map[string]AllowedField{
		"a": {Name: "a", Filterable: true},
		"b": {Name: "b", Filterable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(query.Filters) != 2 || query.Filters[0].Field != "a" || query.Filters[1].Field != "b" {
		t.Fatalf("filters = %#v", query.Filters)
	}
}

func TestParseFilterKeySupportsOperators(t *testing.T) {
	field, operator, ok := parseFilterKey("filter[created_at][gte]")
	if !ok {
		t.Fatal("expected key to parse")
	}
	if field != "created_at" || operator != "gte" {
		t.Fatalf("field=%q operator=%q", field, operator)
	}
}

func TestParseRejectsMalformedFilterKey(t *testing.T) {
	_, err := Parse(url.Values{"filter[role": {"admin"}}, map[string]AllowedField{
		"role": {Name: "role", Filterable: true},
	})
	if err == nil {
		t.Fatal("expected malformed filter error")
	}
}
