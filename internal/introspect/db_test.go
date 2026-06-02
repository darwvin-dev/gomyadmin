package introspect

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- minimal pgx.Rows mock ---

// mockRows implements pgx.Rows for testing.
type mockRows struct {
	rows    [][]any
	pos     int
	scanErr error
	err     error
}

func (m *mockRows) Close()                                    {}
func (m *mockRows) Err() error                               { return m.err }
func (m *mockRows) CommandTag() pgconn.CommandTag            { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRows) Next() bool {
	if m.pos < len(m.rows) {
		m.pos++
		return true
	}
	return false
}
func (m *mockRows) Scan(dest ...any) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	row := m.rows[m.pos-1]
	for i, d := range dest {
		if i >= len(row) {
			break
		}
		switch v := d.(type) {
		case *string:
			*v = row[i].(string)
		case *bool:
			*v = row[i].(bool)
		}
	}
	return nil
}
func (m *mockRows) Values() ([]any, error)                   { return nil, nil }
func (m *mockRows) RawValues() [][]byte                      { return nil }
func (m *mockRows) Conn() *pgx.Conn                          { return nil }

// --- mock querier ---

type mockQuerier struct {
	// calls is a FIFO queue of responses: each entry is returned for successive Query calls.
	calls []*mockRows
	errs  []error
	pos   int
}

func (mq *mockQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	idx := mq.pos
	mq.pos++
	if idx < len(mq.errs) && mq.errs[idx] != nil {
		return nil, mq.errs[idx]
	}
	if idx < len(mq.calls) {
		return mq.calls[idx], nil
	}
	return &mockRows{}, nil
}

// --- tests ---

func TestLoadReturnsErrorOnInvalidDSN(t *testing.T) {
	_, err := Load(context.Background(), "not-a-valid-dsn://???")
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

func TestLoadSchemaQueryError(t *testing.T) {
	q := &mockQuerier{
		calls: nil,
		errs:  []error{errors.New("query failed")},
	}
	_, err := loadSchema(context.Background(), q)
	if err == nil || err.Error() != "query failed" {
		t.Fatalf("expected 'query failed', got %v", err)
	}
}

func TestLoadSchemaEmptyResult(t *testing.T) {
	q := &mockQuerier{
		calls: []*mockRows{
			{}, // tables query returns no rows
		},
	}
	schema, err := loadSchema(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Tables) != 0 {
		t.Fatalf("expected 0 tables, got %d", len(schema.Tables))
	}
}

func TestLoadSchemaSingleTable(t *testing.T) {
	q := &mockQuerier{
		calls: []*mockRows{
			// tables query: one table "public.users"
			{rows: [][]any{{"public", "users"}}},
			// loadColumns query: two columns
			{rows: [][]any{
				{"id", "integer", false, "", true},
				{"email", "text", false, "", false},
			}},
			// loadPrimaryKey query: one PK column
			{rows: [][]any{{"id"}}},
		},
	}
	schema, err := loadSchema(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}
	tbl := schema.Tables[0]
	if tbl.Schema != "public" || tbl.Name != "users" {
		t.Fatalf("unexpected table %q.%q", tbl.Schema, tbl.Name)
	}
	if len(tbl.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(tbl.Columns))
	}
	if len(tbl.PrimaryKey) != 1 || tbl.PrimaryKey[0] != "id" {
		t.Fatalf("unexpected primary key %v", tbl.PrimaryKey)
	}
}

func TestLoadSchemaColumnsQueryError(t *testing.T) {
	q := &mockQuerier{
		calls: []*mockRows{
			{rows: [][]any{{"public", "users"}}},
		},
		errs: []error{nil, errors.New("columns query failed")},
	}
	_, err := loadSchema(context.Background(), q)
	if err == nil || err.Error() != "columns query failed" {
		t.Fatalf("expected 'columns query failed', got %v", err)
	}
}

func TestLoadSchemaPrimaryKeyQueryError(t *testing.T) {
	q := &mockQuerier{
		calls: []*mockRows{
			{rows: [][]any{{"public", "users"}}},
			{rows: [][]any{{"id", "integer", false, "", true}}},
		},
		errs: []error{nil, nil, errors.New("pk query failed")},
	}
	_, err := loadSchema(context.Background(), q)
	if err == nil || err.Error() != "pk query failed" {
		t.Fatalf("expected 'pk query failed', got %v", err)
	}
}

func TestLoadSchemaTableScanError(t *testing.T) {
	q := &mockQuerier{
		calls: []*mockRows{
			{rows: [][]any{{"public", "users"}}, scanErr: errors.New("scan failed")},
		},
	}
	_, err := loadSchema(context.Background(), q)
	if err == nil || err.Error() != "scan failed" {
		t.Fatalf("expected 'scan failed', got %v", err)
	}
}
