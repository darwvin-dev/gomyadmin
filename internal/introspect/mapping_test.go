package introspect

import (
	"testing"

	"github.com/darwvin/gomyadmin/pkg/admin"
)

func TestColumnToFieldType(t *testing.T) {
	cases := []struct {
		dataType string
		want     admin.FieldType
	}{
		{"uuid", admin.FieldUUID},
		{"UUID", admin.FieldUUID},
		{"boolean", admin.FieldBoolean},
		{"bool", admin.FieldBoolean},
		{"integer", admin.FieldInteger},
		{"int", admin.FieldInteger},
		{"int4", admin.FieldInteger},
		{"bigint", admin.FieldInteger},
		{"int8", admin.FieldInteger},
		{"smallint", admin.FieldInteger},
		{"real", admin.FieldFloat},
		{"float4", admin.FieldFloat},
		{"double precision", admin.FieldFloat},
		{"float8", admin.FieldFloat},
		{"numeric", admin.FieldDecimal},
		{"decimal", admin.FieldDecimal},
		{"money", admin.FieldMoney},
		{"text", admin.FieldText},
		{"character varying", admin.FieldString},
		{"varchar", admin.FieldString},
		{"character", admin.FieldString},
		{"char", admin.FieldString},
		{"date", admin.FieldDate},
		{"time", admin.FieldTime},
		{"time without time zone", admin.FieldTime},
		{"timestamp", admin.FieldDateTime},
		{"timestamp without time zone", admin.FieldDateTime},
		{"timestamp with time zone", admin.FieldDateTime},
		{"timestamptz", admin.FieldDateTime},
		{"json", admin.FieldJSON},
		{"jsonb", admin.FieldJSONB},
		{"unknown_type", admin.FieldString},
	}

	for _, c := range cases {
		if got := ColumnToFieldType(c.dataType); got != c.want {
			t.Errorf("ColumnToFieldType(%q) = %q, want %q", c.dataType, got, c.want)
		}
	}
}

func TestSuggestFieldTypeEmail(t *testing.T) {
	col := Column{Name: "email", DataType: "character varying"}
	if got := SuggestFieldType(col); got != admin.FieldEmail {
		t.Errorf("got %q, want email", got)
	}
}

func TestSuggestFieldTypeUserEmail(t *testing.T) {
	col := Column{Name: "user_email", DataType: "text"}
	if got := SuggestFieldType(col); got != admin.FieldEmail {
		t.Errorf("got %q", got)
	}
}

func TestSuggestFieldTypePassword(t *testing.T) {
	col := Column{Name: "password_hash", DataType: "text"}
	if got := SuggestFieldType(col); got != admin.FieldPassword {
		t.Errorf("got %q", got)
	}
}

func TestIsSearchable(t *testing.T) {
	searchable := []Column{
		{Name: "name", DataType: "character varying"},
		{Name: "bio", DataType: "text"},
		{Name: "email", DataType: "character varying"},
	}
	notSearchable := []Column{
		{Name: "created_at", DataType: "timestamp"},
		{Name: "price", DataType: "numeric"},
		{Name: "active", DataType: "boolean"},
	}
	for _, col := range searchable {
		if !IsSearchable(col) {
			t.Errorf("%s (%s) should be searchable", col.Name, col.DataType)
		}
	}
	for _, col := range notSearchable {
		if IsSearchable(col) {
			t.Errorf("%s (%s) should not be searchable", col.Name, col.DataType)
		}
	}
}

func TestIsSortable(t *testing.T) {
	sortable := []Column{
		{Name: "name", DataType: "character varying"},
		{Name: "created_at", DataType: "timestamp"},
		{Name: "amount", DataType: "numeric"},
		{Name: "active", DataType: "boolean"},
	}
	notSortable := []Column{
		{Name: "meta", DataType: "jsonb"},
		{Name: "tags", DataType: "json"},
	}
	for _, col := range sortable {
		if !IsSortable(col) {
			t.Errorf("%s (%s) should be sortable", col.Name, col.DataType)
		}
	}
	for _, col := range notSortable {
		if IsSortable(col) {
			t.Errorf("%s (%s) should not be sortable", col.Name, col.DataType)
		}
	}
}

func TestIsLikelyPrimaryKey(t *testing.T) {
	if !IsLikelyPrimaryKey(Column{Name: "id", DataType: "integer"}) {
		t.Error("id should be likely primary key")
	}
	if !IsLikelyPrimaryKey(Column{Name: "uuid", DataType: "uuid", IsIdentity: false}) {
		t.Error("uuid column should be likely primary key")
	}
	if !IsLikelyPrimaryKey(Column{Name: "created_at", DataType: "timestamp", IsIdentity: true}) {
		t.Error("identity column should be likely primary key")
	}
	if IsLikelyPrimaryKey(Column{Name: "name", DataType: "text"}) {
		t.Error("name should not be primary key")
	}
}
