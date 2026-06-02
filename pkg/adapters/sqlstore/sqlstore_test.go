package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
	_ "modernc.org/sqlite"
)

type testItem struct{}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// Use a file-based DB so multiple connections in the pool share the same tables.
	// (:memory: with pool creates isolated DBs per connection, causing deadlocks in DeleteMany.)
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, q := range []string{
		`CREATE TABLE items (
			id TEXT PRIMARY KEY,
			name TEXT,
			created_at TEXT,
			updated_at TEXT
		)`,
		`CREATE TABLE gomyadmin_audit_logs (
			id TEXT PRIMARY KEY,
			actor_id TEXT, actor_email TEXT, tenant_id TEXT,
			action TEXT, resource TEXT, resource_id TEXT,
			old_values TEXT, new_values TEXT,
			ip_address TEXT, user_agent TEXT, request_id TEXT,
			created_at TEXT
		)`,
		`CREATE TABLE gomyadmin_files (
			id TEXT PRIMARY KEY,
			tenant_id TEXT, key TEXT, name TEXT,
			content_type TEXT, size INTEGER, visibility TEXT,
			created_at TEXT
		)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db := newTestDB(t)
	app := admin.New("Test")
	app.Resource(testItem{}).TableName("items").
		Field("ID").String().Primary().
		Field("Name").String().Searchable().Sortable().Filterable().
		Field("CreatedAt").DateTime().Readonly().
		Field("UpdatedAt").DateTime().Readonly()
	return New(db, app, DialectSQLite)
}

func TestSQLiteUnknownTableReturnsNotFound(t *testing.T) {
	store := newTestStore(t)
	_, _, err := store.List(context.Background(), "missing", "", "super_admin", "", "", nil, 1, 10)
	if err != server.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteCreateAndGet(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	rec, err := store.Create(ctx, "items", "t1", server.Record{"id": "item-1", "name": "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(rec["id"]) != "item-1" {
		t.Fatalf("id = %v", rec["id"])
	}

	got, err := store.Get(ctx, "items", "item-1", "t1", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got["name"]) != "Alice" {
		t.Fatalf("name = %v", got["name"])
	}
}

func TestSQLiteGetMissingReturnsNotFound(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Get(context.Background(), "items", "no-such-id", "", "super_admin")
	if err != server.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteList(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	for i, name := range []string{"Alpha", "Beta", "Gamma"} {
		if _, err := store.Create(ctx, "items", "", server.Record{"id": fmt.Sprintf("id-%d", i), "name": name}); err != nil {
			t.Fatal(err)
		}
	}

	records, total, err := store.List(ctx, "items", "", "super_admin", "", "", nil, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(records) != 3 {
		t.Fatalf("len(records) = %d, want 3", len(records))
	}
}

func TestSQLiteListSearch(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	for i, name := range []string{"Apple", "Banana", "Apricot"} {
		if _, err := store.Create(ctx, "items", "", server.Record{"id": fmt.Sprintf("id-%d", i), "name": name}); err != nil {
			t.Fatal(err)
		}
	}

	records, total, err := store.List(ctx, "items", "", "super_admin", "Ap", "", nil, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("search total = %d, want 2", total)
	}
	if len(records) != 2 {
		t.Fatalf("search records = %d, want 2", len(records))
	}
}

func TestSQLiteListPagination(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	for i := 0; i < 5; i++ {
		if _, err := store.Create(ctx, "items", "", server.Record{"id": fmt.Sprintf("pg-%d", i), "name": fmt.Sprintf("item%d", i)}); err != nil {
			t.Fatal(err)
		}
	}

	records, total, err := store.List(ctx, "items", "", "super_admin", "", "", nil, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(records) != 2 {
		t.Fatalf("page 1 len = %d, want 2", len(records))
	}
}

func TestSQLiteUpdate(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	if _, err := store.Create(ctx, "items", "", server.Record{"id": "upd-1", "name": "Before"}); err != nil {
		t.Fatal(err)
	}

	old, updated, err := store.Update(ctx, "items", "upd-1", "", "super_admin", server.Record{"name": "After"})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(old["name"]) != "Before" {
		t.Fatalf("old name = %v", old["name"])
	}
	if fmt.Sprint(updated["name"]) != "After" {
		t.Fatalf("updated name = %v", updated["name"])
	}
}

func TestSQLiteDelete(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	if _, err := store.Create(ctx, "items", "", server.Record{"id": "del-1", "name": "ToDelete"}); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.Delete(ctx, "items", "del-1", "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(deleted["name"]) != "ToDelete" {
		t.Fatalf("deleted name = %v", deleted["name"])
	}

	_, err = store.Get(ctx, "items", "del-1", "", "super_admin")
	if err != server.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSQLiteDeleteMany(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	ids := []string{"dm-1", "dm-2", "dm-3"}
	for i, name := range []string{"X", "Y", "Z"} {
		if _, err := store.Create(ctx, "items", "", server.Record{"id": ids[i], "name": name}); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := store.DeleteMany(ctx, "items", ids[:2], "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 2 {
		t.Fatalf("deleted count = %d, want 2", len(deleted))
	}

	_, total, err := store.List(ctx, "items", "", "super_admin", "", "", nil, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("remaining total = %d, want 1", total)
	}
}

func TestSQLiteRecordAuditAndAudit(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	store.RecordAudit(ctx, server.AuditEvent{
		ID:         "audit-1",
		ActorID:    "u1",
		ActorEmail: "admin@example.com",
		Action:     "create",
		Resource:   "items",
		ResourceID: "item1",
		CreatedAt:  time.Now().UTC(),
	})

	events, err := store.Audit(ctx, "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected audit events")
	}
	if events[0].ActorEmail != "admin@example.com" {
		t.Fatalf("actor email = %q", events[0].ActorEmail)
	}
}

func TestSQLiteAddFileAndFiles(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	err := store.AddFile(ctx, server.Record{
		"id":           "f-001",
		"key":          "uploads/test.png",
		"name":         "test.png",
		"content_type": "image/png",
		"size":         1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	files, err := store.Files(ctx, "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("files count = %d, want 1", len(files))
	}
	if fmt.Sprint(files[0]["name"]) != "test.png" {
		t.Fatalf("file name = %v", files[0]["name"])
	}
}

func TestSQLiteFileKey(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	err := store.AddFile(ctx, server.Record{
		"id":           "file-001",
		"key":          "uploads/doc.pdf",
		"name":         "doc.pdf",
		"content_type": "application/pdf",
		"size":         2048,
	})
	if err != nil {
		t.Fatal(err)
	}

	key, err := store.FileKey(ctx, "file-001", "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if key != "uploads/doc.pdf" {
		t.Fatalf("key = %q", key)
	}
}

func TestSQLiteFileKeyMissingReturnsNotFound(t *testing.T) {
	store := newTestStore(t)
	_, err := store.FileKey(context.Background(), "no-such-file", "", "super_admin")
	if err != server.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteFilterOperators(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	for i, name := range []string{"Aaron", "Bob", "Anna"} {
		if _, err := store.Create(ctx, "items", "", server.Record{"id": fmt.Sprintf("fo-%d", i), "name": name}); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		filter map[string]string
		want   int
	}{
		{map[string]string{"name::eq": "Bob"}, 1},
		{map[string]string{"name::contains": "an"}, 1},
		{map[string]string{"name::starts_with": "A"}, 2},
		{map[string]string{"name::ends_with": "n"}, 1},
	}

	for _, tt := range tests {
		records, _, err := store.List(ctx, "items", "", "super_admin", "", "", tt.filter, 1, 10)
		if err != nil {
			t.Fatalf("filter %v: %v", tt.filter, err)
		}
		if len(records) != tt.want {
			t.Fatalf("filter %v: got %d records, want %d", tt.filter, len(records), tt.want)
		}
	}
}

func TestSQLiteInvalidFilterReturnsError(t *testing.T) {
	store := newTestStore(t)
	_, _, err := store.List(context.Background(), "items", "", "super_admin", "", "", map[string]string{"id::unsupported": "x"}, 1, 10)
	if err == nil {
		t.Fatal("expected error for unsupported filter operator")
	}
}

func TestSQLiteDialectMySQL(t *testing.T) {
	db := newTestDB(t)
	app := admin.New("Test")
	app.Resource(testItem{}).TableName("items").
		Field("ID").String().Primary().
		Field("Name").String()
	store := New(db, app, DialectMySQL)
	if store == nil {
		t.Fatal("expected non-nil store for MySQL dialect")
	}
}
