package migrate

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"
)

func TestFromFSOrdersMigrationsAndChecksums(t *testing.T) {
	files := fstest.MapFS{
		"migrations/002_add_users.sql": {Data: []byte("create table users(id text);")},
		"migrations/001_init.sql":      {Data: []byte("create table tenants(id text);")},
		"migrations/README.md":         {Data: []byte("ignored")},
	}
	migrations, err := FromFS(files, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 2 {
		t.Fatalf("len = %d", len(migrations))
	}
	if migrations[0].Version != "001" || migrations[1].Version != "002" {
		t.Fatalf("versions = %#v", migrations)
	}
	if migrations[0].Checksum == "" || migrations[0].Checksum == migrations[1].Checksum {
		t.Fatalf("bad checksums: %#v", migrations)
	}
}

func TestUpReturnsErrorWithVersionOnBadSQL(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	runner := Runner{DB: db, Dialect: DialectSQLite}
	migrations := []Migration{
		{Version: "001", Name: "bad", SQL: "NOT VALID SQL !!!!", Checksum: "abc"},
	}
	err = runner.Up(context.Background(), migrations)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "001") {
		t.Fatalf("error should contain version '001', got: %v", err)
	}
}

func newSQLiteRunner(t *testing.T) Runner {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "migrate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return Runner{DB: db, Dialect: DialectSQLite}
}

func TestRunnerUpWithNoMigrations(t *testing.T) {
	runner := newSQLiteRunner(t)
	if err := runner.Up(context.Background(), nil); err != nil {
		t.Fatalf("Up with nil migrations: %v", err)
	}
	if err := runner.Up(context.Background(), []Migration{}); err != nil {
		t.Fatalf("Up with empty migrations: %v", err)
	}
}

func TestRunnerUpAppliesMigrations(t *testing.T) {
	runner := newSQLiteRunner(t)
	migrations := []Migration{
		{Version: "001", Name: "init", SQL: "CREATE TABLE items (id TEXT PRIMARY KEY)", Checksum: "c1"},
		{Version: "002", Name: "add_name", SQL: "ALTER TABLE items ADD COLUMN name TEXT", Checksum: "c2"},
	}
	if err := runner.Up(context.Background(), migrations); err != nil {
		t.Fatalf("Up: %v", err)
	}
	_, err := runner.DB.Exec("INSERT INTO items (id, name) VALUES ('1', 'Alice')")
	if err != nil {
		t.Fatalf("insert after migration: %v", err)
	}
}

func TestRunnerUpIsIdempotent(t *testing.T) {
	runner := newSQLiteRunner(t)
	migrations := []Migration{
		{Version: "001", Name: "init", SQL: "CREATE TABLE things (id TEXT PRIMARY KEY)", Checksum: "abc123"},
	}
	if err := runner.Up(context.Background(), migrations); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	if err := runner.Up(context.Background(), migrations); err != nil {
		t.Fatalf("second Up (idempotent): %v", err)
	}
}

func TestRunnerUpChecksumMismatchReturnsError(t *testing.T) {
	runner := newSQLiteRunner(t)
	first := []Migration{
		{Version: "001", Name: "init", SQL: "CREATE TABLE a (id TEXT)", Checksum: "original-checksum"},
	}
	if err := runner.Up(context.Background(), first); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	second := []Migration{
		{Version: "001", Name: "init", SQL: "CREATE TABLE a (id TEXT)", Checksum: "tampered-checksum"},
	}
	err := runner.Up(context.Background(), second)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("error should mention checksum, got: %v", err)
	}
}

func TestRunnerUpBadSQLRecordsNotApplied(t *testing.T) {
	runner := newSQLiteRunner(t)
	bad := []Migration{
		{Version: "001", Name: "bad", SQL: "THIS IS NOT VALID SQL", Checksum: "x"},
	}
	if err := runner.Up(context.Background(), bad); err == nil {
		t.Fatal("expected error for invalid SQL")
	}
	// After failure, re-apply with a corrected migration at same version should succeed
	// (the failed migration was rolled back, so version 001 is not in the table)
	fixed := []Migration{
		{Version: "001", Name: "fixed", SQL: "CREATE TABLE ok (id TEXT)", Checksum: "y"},
	}
	if err := runner.Up(context.Background(), fixed); err != nil {
		t.Fatalf("re-apply after rollback failed: %v", err)
	}
}

func TestRunnerRequiresDB(t *testing.T) {
	runner := Runner{}
	err := runner.Up(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when DB is nil")
	}
}

func TestFromFSIgnoresNonSQLFiles(t *testing.T) {
	files := fstest.MapFS{
		"m/001_init.sql": {Data: []byte("CREATE TABLE a (id TEXT)")},
		"m/README.md":    {Data: []byte("docs")},
	}
	migrations, err := FromFS(files, "m")
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(migrations))
	}
}

func TestFromFSPreservesVersionAndName(t *testing.T) {
	files := fstest.MapFS{
		"m/003_create_users.sql": {Data: []byte("CREATE TABLE users (id TEXT)")},
	}
	migrations, err := FromFS(files, "m")
	if err != nil {
		t.Fatal(err)
	}
	if migrations[0].Version != "003" {
		t.Fatalf("version = %q", migrations[0].Version)
	}
	if migrations[0].Name != "create_users" {
		t.Fatalf("name = %q", migrations[0].Name)
	}
}

func TestRunnerCustomTable(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "ct.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	runner := Runner{DB: db, Dialect: DialectSQLite, Table: "my_migrations"}
	migrations := []Migration{
		{Version: "001", Name: "init", SQL: "CREATE TABLE x (id TEXT)", Checksum: "c"},
	}
	if err := runner.Up(context.Background(), migrations); err != nil {
		t.Fatalf("Up with custom table: %v", err)
	}
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM my_migrations").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 row in my_migrations, got %d", count)
	}
}
