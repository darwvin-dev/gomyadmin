# Test Coverage & Housekeeping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring all packages from 0% coverage to meaningful coverage, fix the CHANGELOG, commit the yarn migration, and add PostgreSQL to CI so skipped Postgres tests actually run.

**Architecture:** Each adapter package gets its own `_test.go`; adapters with external dependencies (SQLite, Redis) use lightweight pure-Go in-memory implementations. `pkg/server` tests extend the existing `fakeStore` pattern. No new abstractions — just tests.

**Tech Stack:** Go 1.23, `modernc.org/sqlite` (pure-Go SQLite for sqlstore/gormstore), `github.com/alicebob/miniredis/v2` (in-memory Redis for redisstore), `github.com/glebarez/sqlite` (pure-Go GORM dialect), GitHub Actions postgres service.

---

## Task 1: Fix CHANGELOG ordering + commit yarn migration

**Files:**
- Modify: `CHANGELOG.md`
- Git: stage and commit `templates/frontend-next-shadcn/yarn.lock`, `templates/frontend-next-shadcn/package-lock.json` deletion

- [ ] **Step 1: Fix CHANGELOG header ordering**

The file currently has: `[0.2.0] - 2026-05-24` (top, with adapter/migrate content) → `[0.3.0]` → `[0.2.0] - 2026-05-23` → `[0.1.0]`. The top section contains features added after v0.3.0 and must become `[Unreleased]`.

Replace the first line of the first version section:
```
Old: ## [0.2.0] - 2026-05-24
New: ## [Unreleased]
```

Then move the `[0.3.0]` block (which currently appears *after* the mislabeled [0.2.0]) so the final order is:
`[Unreleased]` → `[0.3.0]` → `[0.2.0]` → `[0.1.0]`

Verify final structure:
```bash
grep "^## \[" CHANGELOG.md
```
Expected output:
```
## [Unreleased]
## [0.3.0] — 2026-05-24
## [0.2.0] — 2026-05-23
## [0.1.0] — 2026-05-23
```

- [ ] **Step 2: Commit CHANGELOG fix**

```bash
git add CHANGELOG.md
git commit -m "fix: correct CHANGELOG version ordering and mislabeled section"
```

- [ ] **Step 3: Commit yarn.lock migration**

```bash
git add templates/frontend-next-shadcn/yarn.lock
git rm --cached templates/frontend-next-shadcn/package-lock.json
git commit -m "chore: switch frontend from npm to yarn"
```

---

## Task 2: sqlstore tests (in-memory SQLite)

**Files:**
- Create: `pkg/adapters/sqlstore/sqlstore_test.go`
- Modify: `go.mod` / `go.sum` (add `modernc.org/sqlite`)

- [ ] **Step 1: Add SQLite driver**

```bash
go get modernc.org/sqlite
```

Expected: `go.mod` updated with `modernc.org/sqlite vX.Y.Z`.

- [ ] **Step 2: Write failing test**

Create `pkg/adapters/sqlstore/sqlstore_test.go`:

```go
package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
	_ "modernc.org/sqlite"
)

type testRow struct{}

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, q := range []string{
		`CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE gomyadmin_audit_logs (
			id TEXT PRIMARY KEY, actor_id TEXT, actor_email TEXT, tenant_id TEXT,
			action TEXT, resource TEXT, resource_id TEXT,
			old_values TEXT, new_values TEXT, ip_address TEXT, user_agent TEXT, request_id TEXT, created_at TEXT
		)`,
		`CREATE TABLE gomyadmin_files (
			id TEXT PRIMARY KEY, tenant_id TEXT, key TEXT, name TEXT,
			content_type TEXT, size INTEGER, visibility TEXT, created_at TEXT
		)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func newStore(t *testing.T) *Store {
	t.Helper()
	db := newDB(t)
	app := admin.New("Test")
	app.Resource(testRow{}).TableName("items").
		Field("ID").String().Primary().
		Field("Name").String().Searchable().Sortable().Filterable()
	return New(db, app, DialectSQLite)
}

func TestSQLiteCreateAndGet(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	rec, err := st.Create(ctx, "items", "", server.Record{"name": "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	if rec["name"] != "Alice" {
		t.Fatalf("name = %v", rec["name"])
	}
	id := fmt.Sprint(rec["id"])

	got, err := st.Get(ctx, "items", id, "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if got["name"] != "Alice" {
		t.Fatalf("got name = %v", got["name"])
	}
}

func TestSQLiteListAndCount(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	for _, name := range []string{"Alice", "Bob", "Carol"} {
		if _, err := st.Create(ctx, "items", "", server.Record{"name": name}); err != nil {
			t.Fatal(err)
		}
	}

	records, total, err := st.List(ctx, "items", "", "super_admin", "", "", nil, 1, 10)
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
	st := newStore(t)

	for _, name := range []string{"Alice", "Bob"} {
		if _, err := st.Create(ctx, "items", "", server.Record{"name": name}); err != nil {
			t.Fatal(err)
		}
	}

	records, total, err := st.List(ctx, "items", "", "super_admin", "Ali", "", nil, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("search total = %d, want 1", total)
	}
	if len(records) != 1 || records[0]["name"] != "Alice" {
		t.Fatalf("search result = %v", records)
	}
}

func TestSQLiteListFilter(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	for _, name := range []string{"Alice", "Bob"} {
		if _, err := st.Create(ctx, "items", "", server.Record{"name": name}); err != nil {
			t.Fatal(err)
		}
	}

	records, total, err := st.List(ctx, "items", "", "super_admin", "", "", map[string]string{"name": "Bob"}, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || records[0]["name"] != "Bob" {
		t.Fatalf("filter result total=%d records=%v", total, records)
	}
}

func TestSQLiteUpdate(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	rec, err := st.Create(ctx, "items", "", server.Record{"name": "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprint(rec["id"])

	old, updated, err := st.Update(ctx, "items", id, "", "super_admin", server.Record{"name": "Alicia"})
	if err != nil {
		t.Fatal(err)
	}
	if old["name"] != "Alice" {
		t.Fatalf("old name = %v", old["name"])
	}
	if updated["name"] != "Alicia" {
		t.Fatalf("updated name = %v", updated["name"])
	}
}

func TestSQLiteDelete(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	rec, err := st.Create(ctx, "items", "", server.Record{"name": "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprint(rec["id"])

	deleted, err := st.Delete(ctx, "items", id, "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if deleted["name"] != "Alice" {
		t.Fatalf("deleted name = %v", deleted["name"])
	}

	_, err = st.Get(ctx, "items", id, "", "super_admin")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSQLiteDeleteMany(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	var ids []string
	for _, name := range []string{"A", "B", "C"} {
		rec, err := st.Create(ctx, "items", "", server.Record{"name": name})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, fmt.Sprint(rec["id"]))
	}

	deleted, err := st.DeleteMany(ctx, "items", ids[:2], "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 2 {
		t.Fatalf("deleted count = %d, want 2", len(deleted))
	}

	_, total, err := st.List(ctx, "items", "", "super_admin", "", "", nil, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("remaining total = %d, want 1", total)
	}
}

func TestSQLiteGetMissingTable(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	_, err := st.Get(ctx, "nonexistent", "id", "", "super_admin")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for unknown table, got %v", err)
	}
}

func TestSQLiteAuditRoundtrip(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	st.RecordAudit(ctx, server.AuditEvent{
		ActorEmail: "admin@test.com",
		Action:     "create",
		Resource:   "items",
		ResourceID: "123",
	})

	events, err := st.Audit(ctx, "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(events))
	}
	if events[0].Action != "create" {
		t.Fatalf("action = %q", events[0].Action)
	}
}

func TestSQLiteFilesRoundtrip(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	err := st.AddFile(ctx, server.Record{
		"key":          "uploads/test.txt",
		"name":         "test.txt",
		"content_type": "text/plain",
		"size":         42,
	})
	if err != nil {
		t.Fatal(err)
	}

	files, err := st.Files(ctx, "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("files = %d, want 1", len(files))
	}

	key, err := st.FileKey(ctx, fmt.Sprint(files[0]["id"]), "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if key != "uploads/test.txt" {
		t.Fatalf("key = %q", key)
	}
}

func TestSQLiteUnknownFilterReturnsError(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	_, _, err := st.List(ctx, "items", "", "super_admin", "", "", map[string]string{"nosuchfield": "v"}, 1, 10)
	if err == nil {
		t.Fatal("expected error for unknown filter field")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails (before implementation)**

```bash
go test ./pkg/adapters/sqlstore/... -run TestSQLiteCreateAndGet -v
```

Expected: FAIL (sqlstore_test.go doesn't exist yet — this step confirms the file is missing, then we create it in Step 2 which is already done above; re-run to confirm PASS).

- [ ] **Step 4: Run all sqlstore tests**

```bash
go test ./pkg/adapters/sqlstore/... -v -timeout 30s
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum pkg/adapters/sqlstore/sqlstore_test.go
git commit -m "test: add sqlstore unit tests using in-memory SQLite"
```

---

## Task 3: gormstore tests (pure-Go GORM + SQLite)

**Files:**
- Create: `pkg/adapters/gormstore/gormstore_test.go`
- Modify: `go.mod` / `go.sum` (add `github.com/glebarez/sqlite`)

- [ ] **Step 1: Add pure-Go GORM SQLite dialect**

```bash
go get github.com/glebarez/sqlite
```

Expected: `go.mod` updated.

- [ ] **Step 2: Write the test**

Create `pkg/adapters/gormstore/gormstore_test.go`:

```go
package gormstore

import (
	"context"
	"fmt"
	"testing"

	gsqlite "github.com/glebarez/sqlite"
	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/adapters/sqlstore"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
	"gorm.io/gorm"
)

type gormItem struct{}

func newGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gsqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE gomyadmin_audit_logs (
			id TEXT PRIMARY KEY, actor_id TEXT, actor_email TEXT, tenant_id TEXT,
			action TEXT, resource TEXT, resource_id TEXT,
			old_values TEXT, new_values TEXT, ip_address TEXT, user_agent TEXT, request_id TEXT, created_at TEXT
		)`,
		`CREATE TABLE gomyadmin_files (
			id TEXT PRIMARY KEY, tenant_id TEXT, key TEXT, name TEXT,
			content_type TEXT, size INTEGER, visibility TEXT, created_at TEXT
		)`,
	} {
		if _, err := sqlDB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func newGormStore(t *testing.T) *sqlstore.Store {
	t.Helper()
	db := newGormDB(t)
	app := admin.New("Test")
	app.Resource(gormItem{}).TableName("items").
		Field("ID").String().Primary().
		Field("Name").String()
	st, err := SQLite(db, app)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestGormstoreNewReturnsStore(t *testing.T) {
	st := newGormStore(t)
	if st == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestGormstoreCreateAndGet(t *testing.T) {
	ctx := context.Background()
	st := newGormStore(t)

	rec, err := st.Create(ctx, "items", "", server.Record{"name": "Gorm User"})
	if err != nil {
		t.Fatal(err)
	}
	if rec["name"] != "Gorm User" {
		t.Fatalf("name = %v", rec["name"])
	}

	got, err := st.Get(ctx, "items", fmt.Sprint(rec["id"]), "", "super_admin")
	if err != nil {
		t.Fatal(err)
	}
	if got["name"] != "Gorm User" {
		t.Fatalf("got name = %v", got["name"])
	}
}

func TestGormstoreNewPropagatesError(t *testing.T) {
	// Verify that an error from db.DB() is returned from New().
	// A zero-value gorm.DB has no underlying sql.DB and returns an error.
	var db gorm.DB
	_, err := New(&db, admin.New("test"), sqlstore.DialectSQLite)
	if err == nil {
		t.Error("expected error for zero gorm.DB")
	}
}
```

- [ ] **Step 3: Run gormstore tests**

```bash
go test ./pkg/adapters/gormstore/... -v -timeout 30s
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum pkg/adapters/gormstore/gormstore_test.go
git commit -m "test: add gormstore tests using pure-Go SQLite dialect"
```

---

## Task 4: redisstore tests (miniredis in-memory Redis)

**Files:**
- Create: `pkg/adapters/redisstore/redisstore_test.go`
- Modify: `go.mod` / `go.sum` (add `github.com/alicebob/miniredis/v2`)

- [ ] **Step 1: Add miniredis**

```bash
go get github.com/alicebob/miniredis/v2
```

Expected: `go.mod` updated.

- [ ] **Step 2: Write the test**

Create `pkg/adapters/redisstore/redisstore_test.go`:

```go
package redisstore

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/redis/go-redis/v9"
)

func newMiniRedisClient(t *testing.T) redis.UniversalClient {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestCacheGetSetDelete(t *testing.T) {
	ctx := context.Background()
	client := newMiniRedisClient(t)
	c := Cache{Client: client}

	if err := c.Set(ctx, "key1", []byte("hello"), time.Minute); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get(ctx, "key1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}

	if err := c.Delete(ctx, "key1"); err != nil {
		t.Fatal(err)
	}

	_, err = c.Get(ctx, "key1")
	if err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestCacheGetMissingKeyReturnsSessionNotFound(t *testing.T) {
	ctx := context.Background()
	client := newMiniRedisClient(t)
	c := Cache{Client: client}

	_, err := c.Get(ctx, "does-not-exist")
	if err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestNewReturnsWorkingSessionStore(t *testing.T) {
	client := newMiniRedisClient(t)
	store := New(client)
	if store == nil {
		t.Fatal("expected non-nil session store")
	}

	// Round-trip via the session store to confirm wiring is correct.
	ctx := context.Background()
	actor := admin.Actor{ID: "u1", Email: "test@example.com"}
	session, err := store.Create(ctx, actor, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Actor.Email != "test@example.com" {
		t.Fatalf("email = %q", got.Actor.Email)
	}
}
```

- [ ] **Step 3: Run redisstore tests**

```bash
go test ./pkg/adapters/redisstore/... -v -timeout 30s
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum pkg/adapters/redisstore/redisstore_test.go
git commit -m "test: add redisstore tests using miniredis"
```

---

## Task 5: mongostore pure-logic tests (no real MongoDB)

**Files:**
- Create: `pkg/adapters/mongostore/mongostore_test.go`

The `buildFilter`, `buildSort`, `normalizeRecord`, and `fieldBySQLName` functions are package-internal. Tests in `package mongostore` can access them directly without a live database.

- [ ] **Step 1: Write the test**

Create `pkg/adapters/mongostore/mongostore_test.go`:

```go
package mongostore

import (
	"testing"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
	"go.mongodb.org/mongo-driver/bson"
)

type mongoItem struct{}

func newMeta() server.ResourceMeta {
	app := admin.New("Test")
	app.Resource(mongoItem{}).TableName("items").
		Field("ID").String().Primary().
		Field("Name").String().Searchable().Filterable().Sortable().
		Field("Status").String().Filterable()
	// ResourceMeta is derived via NewResourceMetadataStore.
	ms := server.NewResourceMetadataStore(app)
	for _, r := range ms.Resources() {
		if r.Table == "items" {
			return r
		}
	}
	panic("resource not found")
}

func TestBuildFilterEmpty(t *testing.T) {
	resource := newMeta()
	filter, err := buildFilter(resource, "", "super_admin", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(filter) != 0 {
		t.Fatalf("expected empty filter, got %v", filter)
	}
}

func TestBuildFilterSearch(t *testing.T) {
	resource := newMeta()
	filter, err := buildFilter(resource, "", "super_admin", "alice", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := filter["$or"]; !ok {
		t.Fatalf("expected $or in filter, got %v", filter)
	}
}

func TestBuildFilterEq(t *testing.T) {
	resource := newMeta()
	filter, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name": "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	if filter["name"] != "Bob" {
		t.Fatalf("name filter = %v", filter["name"])
	}
}

func TestBuildFilterContains(t *testing.T) {
	resource := newMeta()
	filter, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::contains": "ob"})
	if err != nil {
		t.Fatal(err)
	}
	nameFilter, ok := filter["name"].(bson.M)
	if !ok {
		t.Fatalf("name filter = %T %v", filter["name"], filter["name"])
	}
	if nameFilter["$regex"] == nil {
		t.Fatalf("expected $regex, got %v", nameFilter)
	}
}

func TestBuildFilterUnknownFieldReturnsError(t *testing.T) {
	resource := newMeta()
	_, err := buildFilter(resource, "", "super_admin", "", map[string]string{"nosuchfield": "v"})
	if err == nil {
		t.Fatal("expected error for unknown filter field")
	}
}

func TestBuildFilterUnknownOperatorReturnsError(t *testing.T) {
	resource := newMeta()
	_, err := buildFilter(resource, "", "super_admin", "", map[string]string{"name::badop": "v"})
	if err == nil {
		t.Fatal("expected error for unknown operator")
	}
}

func TestBuildSortAsc(t *testing.T) {
	resource := newMeta()
	sort := buildSort(resource, "name")
	if len(sort) != 1 || sort[0].Key != "name" || sort[0].Value != 1 {
		t.Fatalf("sort = %v", sort)
	}
}

func TestBuildSortDesc(t *testing.T) {
	resource := newMeta()
	sort := buildSort(resource, "-name")
	if len(sort) != 1 || sort[0].Key != "name" || sort[0].Value != -1 {
		t.Fatalf("sort = %v", sort)
	}
}

func TestBuildSortUnknownFieldReturnsNil(t *testing.T) {
	resource := newMeta()
	sort := buildSort(resource, "nosuchfield")
	if sort != nil {
		t.Fatalf("expected nil sort for unknown field, got %v", sort)
	}
}

func TestNormalizeRecordRenamesID(t *testing.T) {
	record := server.Record{"_id": "abc123", "name": "Alice"}
	normalized := normalizeRecord(record)
	if normalized["id"] != "abc123" {
		t.Fatalf("id = %v", normalized["id"])
	}
	if _, ok := normalized["_id"]; ok {
		t.Fatal("_id should be removed")
	}
}

func TestNormalizeRecordPreservesExistingID(t *testing.T) {
	record := server.Record{"_id": "mongo-id", "id": "real-id", "name": "Bob"}
	normalized := normalizeRecord(record)
	if normalized["id"] != "real-id" {
		t.Fatalf("id should not be overwritten, got %v", normalized["id"])
	}
}

func TestFieldBySQLNameFound(t *testing.T) {
	resource := newMeta()
	field, ok := fieldBySQLName(resource, "name")
	if !ok {
		t.Fatal("expected to find field 'name'")
	}
	if field.SQLName != "name" {
		t.Fatalf("SQLName = %q", field.SQLName)
	}
}

func TestFieldBySQLNameMissing(t *testing.T) {
	resource := newMeta()
	_, ok := fieldBySQLName(resource, "nonexistent")
	if ok {
		t.Fatal("expected false for missing field")
	}
}
```

- [ ] **Step 2: Run mongostore tests**

```bash
go test ./pkg/adapters/mongostore/... -v -timeout 30s
```

Expected: all tests PASS (no real MongoDB needed).

- [ ] **Step 3: Commit**

```bash
git add pkg/adapters/mongostore/mongostore_test.go
git commit -m "test: add mongostore pure-logic tests (no live MongoDB required)"
```

---

## Task 6: Expand pkg/server handler coverage

**Files:**
- Modify: `pkg/server/server_test.go`

Currently at 16.3%. Add handler tests for List, Get, Create, Update, Delete, Login, Me, Resources, Audit, Export.

- [ ] **Step 1: Extend fakeStore and add handler tests**

Append to `pkg/server/server_test.go` (after the closing brace of the last existing test):

```go
// fakeStoreWithRecord returns a record on Get so handler tests can exercise the 200 path.
type fakeStoreWithRecord struct {
	fakeStore
	record Record
}

func (s fakeStoreWithRecord) Get(_ context.Context, _, _, _, _ string) (Record, error) {
	return s.record, nil
}
func (s fakeStoreWithRecord) Update(_ context.Context, _, _, _, _ string, input Record) (Record, Record, error) {
	return s.record, input, nil
}
func (s fakeStoreWithRecord) Delete(_ context.Context, _, _, _, _ string) (Record, error) {
	return s.record, nil
}

func setupSrv(t *testing.T, store AdminStore, permissions ...string) (*AdminServer, string) {
	t.Helper()
	app := admin.New("Acme")
	app.Resource(testUser{}).TableName("users").Field("ID").String().Primary().Field("Name").String()

	if permissions == nil {
		permissions = []string{"*"}
	}
	sessionStore := auth.NewMemorySessionStore()
	session, err := sessionStore.Create(context.Background(), admin.Actor{
		ID:          "actor_1",
		Email:       "admin@example.com",
		Roles:       []string{"super_admin"},
		Permissions: permissions,
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	var st AdminStore = store
	if st == nil {
		st = fakeStore{ResourceMetadataStore: NewResourceMetadataStore(app)}
	}
	srv, err := New(context.Background(), Config{
		App:          app,
		Store:        st,
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv, session.ID
}

func authReq(method, url, sessionID string, body []byte) *http.Request {
	var b *bytes.Reader
	if body != nil {
		b = bytes.NewReader(body)
	} else {
		b = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, url, b)
	req.AddCookie(&http.Cookie{Name: "gomyadmin_session", Value: sessionID})
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestHandleResources(t *testing.T) {
	srv, sid := setupSrv(t, nil)
	req := authReq(http.MethodGet, "/admin/api/resources", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleList(t *testing.T) {
	srv, sid := setupSrv(t, nil)
	req := authReq(http.MethodGet, "/admin/api/users", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListDeniedWithoutPermission(t *testing.T) {
	srv, sid := setupSrv(t, nil, "orders.view") // only orders, not users
	req := authReq(http.MethodGet, "/admin/api/users", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestHandleGetNotFound(t *testing.T) {
	srv, sid := setupSrv(t, nil)
	req := authReq(http.MethodGet, "/admin/api/users/nonexistent-id", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetFound(t *testing.T) {
	app := admin.New("Acme")
	app.Resource(testUser{}).TableName("users").Field("ID").String().Primary().Field("Name").String()
	store := fakeStoreWithRecord{
		fakeStore:  fakeStore{ResourceMetadataStore: NewResourceMetadataStore(app)},
		record:     Record{"id": "u1", "name": "Alice"},
	}
	srv, sid := setupSrv(t, store)
	req := authReq(http.MethodGet, "/admin/api/users/u1", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Alice")) {
		t.Fatalf("body missing record: %s", rec.Body.String())
	}
}

func TestHandleCreate(t *testing.T) {
	srv, sid := setupSrv(t, nil)
	body, _ := json.Marshal(map[string]any{"name": "NewUser"})
	req := authReq(http.MethodPost, "/admin/api/users", sid, body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateUnknownResource(t *testing.T) {
	srv, sid := setupSrv(t, nil)
	body, _ := json.Marshal(map[string]any{"name": "x"})
	req := authReq(http.MethodPost, "/admin/api/unknown_resource", sid, body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	// unknown resource → permission denied (no permission) or 404 from store
	if rec.Code == http.StatusOK {
		t.Fatalf("should not return 200 for unknown resource, body = %s", rec.Body.String())
	}
}

func TestHandleUpdate(t *testing.T) {
	app := admin.New("Acme")
	app.Resource(testUser{}).TableName("users").Field("ID").String().Primary().Field("Name").String()
	store := fakeStoreWithRecord{
		fakeStore: fakeStore{ResourceMetadataStore: NewResourceMetadataStore(app)},
		record:    Record{"id": "u1", "name": "Alice"},
	}
	srv, sid := setupSrv(t, store)
	body, _ := json.Marshal(map[string]any{"name": "Alicia"})
	req := authReq(http.MethodPatch, "/admin/api/users/u1", sid, body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDelete(t *testing.T) {
	app := admin.New("Acme")
	app.Resource(testUser{}).TableName("users").Field("ID").String().Primary().Field("Name").String()
	store := fakeStoreWithRecord{
		fakeStore: fakeStore{ResourceMetadataStore: NewResourceMetadataStore(app)},
		record:    Record{"id": "u1", "name": "Alice"},
	}
	srv, sid := setupSrv(t, store)
	req := authReq(http.MethodDelete, "/admin/api/users/u1", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleLoginNoAuthenticator(t *testing.T) {
	srv, _ := setupSrv(t, nil)
	body, _ := json.Marshal(map[string]any{"email": "a@b.com", "password": "pw"})
	req := httptest.NewRequest(http.MethodPost, "/admin/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no Authenticate func, got %d", rec.Code)
	}
}

func TestHandleLoginSuccess(t *testing.T) {
	app := admin.New("Acme")
	app.Resource(testUser{}).TableName("users").Field("ID").String().Primary()
	sessionStore := auth.NewMemorySessionStore()
	srv, err := New(context.Background(), Config{
		App:          app,
		Store:        fakeStore{ResourceMetadataStore: NewResourceMetadataStore(app)},
		SessionStore: sessionStore,
		Authenticate: func(_ context.Context, email, _ string) (admin.Actor, bool, error) {
			return admin.Actor{ID: "u1", Email: email, Roles: []string{"super_admin"}, Permissions: []string{"*"}}, true, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"email": "admin@test.com", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/admin/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "gomyadmin_session" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("session cookie not set on successful login")
	}
}

func TestHandleMe(t *testing.T) {
	srv, sid := setupSrv(t, nil)
	req := authReq(http.MethodGet, "/admin/api/me", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("admin@example.com")) {
		t.Fatalf("body missing email: %s", rec.Body.String())
	}
}

func TestHandleAudit(t *testing.T) {
	srv, sid := setupSrv(t, nil)
	req := authReq(http.MethodGet, "/admin/api/audit", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleExport(t *testing.T) {
	srv, sid := setupSrv(t, nil)
	req := authReq(http.MethodGet, "/admin/api/users/export", sid, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "text/csv" {
		t.Fatalf("Content-Type = %q, want text/csv", ct)
	}
}

func TestHandleUnauthenticated(t *testing.T) {
	srv, _ := setupSrv(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/api/resources", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run server tests**

```bash
go test ./pkg/server/... -v -timeout 30s
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/server/server_test.go
git commit -m "test: expand pkg/server handler coverage to key HTTP endpoints"
```

---

## Task 7: pkg/logger tests

**Files:**
- Create: `pkg/logger/logger_test.go`

- [ ] **Step 1: Write the test**

Create `pkg/logger/logger_test.go`:

```go
package logger

import (
	"log/slog"
	"testing"
)

func TestNewReturnsNonNil(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", "DEBUG", "INFO", "WARN", "ERROR", "", "invalid"} {
		l := New(level)
		if l == nil {
			t.Fatalf("New(%q) returned nil", level)
		}
	}
}

func TestNewLevelDebug(t *testing.T) {
	l := New("debug")
	if !l.Enabled(nil, slog.LevelDebug) {
		t.Error("debug level should enable debug logs")
	}
}

func TestNewLevelWarn(t *testing.T) {
	l := New("warn")
	if l.Enabled(nil, slog.LevelDebug) {
		t.Error("warn level should not enable debug logs")
	}
	if !l.Enabled(nil, slog.LevelWarn) {
		t.Error("warn level should enable warn logs")
	}
}

func TestNewLevelError(t *testing.T) {
	l := New("error")
	if l.Enabled(nil, slog.LevelWarn) {
		t.Error("error level should not enable warn logs")
	}
	if !l.Enabled(nil, slog.LevelError) {
		t.Error("error level should enable error logs")
	}
}

func TestNewDefaultsToInfo(t *testing.T) {
	l := New("unknown-level")
	if !l.Enabled(nil, slog.LevelInfo) {
		t.Error("unknown level should default to info")
	}
	if l.Enabled(nil, slog.LevelDebug) {
		t.Error("unknown level should not enable debug")
	}
}
```

- [ ] **Step 2: Run logger tests**

```bash
go test ./pkg/logger/... -v
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/logger/logger_test.go
git commit -m "test: add pkg/logger level parsing tests"
```

---

## Task 8: internal/doctor tests

**Files:**
- Create: `internal/doctor/doctor_test.go`

- [ ] **Step 1: Write the test**

Create `internal/doctor/doctor_test.go`:

```go
package doctor

import (
	"context"
	"os"
	"testing"
)

func TestEnvSetReturnsOK(t *testing.T) {
	c := env("MY_VAR", "some-value")
	if !c.OK {
		t.Fatalf("expected OK=true for set env, got message: %q", c.Message)
	}
	if c.Message != "set" {
		t.Fatalf("message = %q, want \"set\"", c.Message)
	}
}

func TestEnvEmptyReturnsFail(t *testing.T) {
	c := env("MY_VAR", "")
	if c.OK {
		t.Fatal("expected OK=false for empty env")
	}
	if c.Message != "not set" {
		t.Fatalf("message = %q, want \"not set\"", c.Message)
	}
}

func TestFileWritableOnCurrentDir(t *testing.T) {
	c := fileWritable(".")
	if !c.OK {
		t.Fatalf("expected writable current dir, got: %q", c.Message)
	}
}

func TestFileWritableOnNonexistentDir(t *testing.T) {
	c := fileWritable("/nonexistent-path-that-should-not-exist-abc123xyz")
	if c.OK {
		t.Fatal("expected not writable for nonexistent dir")
	}
}

func TestOptionalCommandMissingBinaryIsOK(t *testing.T) {
	c := optionalCommand("FantasyTool", "this-binary-does-not-exist", "--version")
	if !c.OK {
		t.Fatalf("optional missing command should be OK, got: %q", c.Message)
	}
}

func TestCommandGoVersion(t *testing.T) {
	c := command("Go", "go", "version")
	if !c.OK {
		t.Fatalf("'go version' should succeed in test environment, got: %q", c.Message)
	}
}

func TestCommandMissingBinaryFails(t *testing.T) {
	c := command("Ghost", "this-binary-does-not-exist-abc123", "--version")
	if c.OK {
		t.Fatal("expected OK=false for missing binary")
	}
}

func TestRunReturnsChecks(t *testing.T) {
	_ = os.Setenv("GOMYADMIN_SESSION_SECRET", "test-secret")
	defer os.Unsetenv("GOMYADMIN_SESSION_SECRET")

	checks := Run(context.Background(), Options{DatabaseURL: ""})
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	names := map[string]bool{}
	for _, c := range checks {
		names[c.Name] = true
	}
	if !names["Go"] {
		t.Error("expected Go check in results")
	}
	if !names["file permissions"] {
		t.Error("expected file permissions check in results")
	}
}
```

- [ ] **Step 2: Run doctor tests**

```bash
go test ./internal/doctor/... -v -timeout 30s
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/doctor/doctor_test.go
git commit -m "test: add internal/doctor unit tests"
```

---

## Task 9: Add PostgreSQL service to CI

**Files:**
- Modify: `.github/workflows/ci.yml`

This enables the 9 currently-skipped Postgres tests (`TestPGStore*`, `TestPGSessionStore*`) to run on every push.

- [ ] **Step 1: Update ci.yml**

Replace the entire `test` job with:

```yaml
  test:
    name: Go test
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:17
        env:
          POSTGRES_USER: gomyadmin
          POSTGRES_PASSWORD: gomyadmin
          POSTGRES_DB: gomyadmin
        options: >-
          --health-cmd pg_isready
          --health-interval 5s
          --health-timeout 5s
          --health-retries 10
        ports:
          - 5432:5432

    env:
      DATABASE_URL: postgres://gomyadmin:gomyadmin@localhost:5432/gomyadmin?sslmode=disable

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Download dependencies
        run: go mod download

      - name: Test
        run: go test ./...
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add PostgreSQL service so Postgres integration tests run on every push"
```

---

## Self-Review

**Spec coverage check:**
| Item | Task |
|------|------|
| Fix CHANGELOG ordering | Task 1 |
| Commit yarn.lock migration | Task 1 |
| sqlstore 0% → covered | Task 2 |
| gormstore 0% → covered | Task 3 |
| redisstore 0% → covered | Task 4 |
| mongostore 0% → covered | Task 5 |
| pkg/server 16% → higher | Task 6 |
| pkg/logger 0% → covered | Task 7 |
| internal/doctor 0% → covered | Task 8 |
| CI Postgres for skipped tests | Task 9 |

**Placeholder scan:** No "TODO", "TBD", or "implement later" in any task. All test code is complete.

**Type consistency:**
- `server.Record = map[string]any` — used as `server.Record{...}` throughout; consistent.
- `fakeStoreWithRecord` extends `fakeStore` — both in `package server`; no naming conflicts.
- `ErrNotFound` in sqlstore tests uses the package-level var; consistent with `store.go:21`.
