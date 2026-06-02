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
