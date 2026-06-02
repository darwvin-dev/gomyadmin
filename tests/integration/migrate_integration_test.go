//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/darwvin-dev/gomyadmin/pkg/migrate"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func pgDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func uniqueTable(prefix string) string {
	// Use a unique table name per test to avoid conflicts
	return fmt.Sprintf("%s_%d", prefix, os.Getpid())
}

func TestMigrateIntegApplyAndIdempotent(t *testing.T) {
	db := pgDB(t)
	migrateTable := uniqueTable("integ_schema_migrations")
	dataTable := uniqueTable("integ_migrate_items")

	t.Cleanup(func() {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %q", dataTable))
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %q", migrateTable))
	})

	runner := migrate.Runner{DB: db, Dialect: migrate.DialectPostgres, Table: migrateTable}
	migrations := []migrate.Migration{
		{
			Version:  "001",
			Name:     "create",
			SQL:      fmt.Sprintf("CREATE TABLE IF NOT EXISTS %q (id TEXT PRIMARY KEY)", dataTable),
			Checksum: "c1",
		},
		{
			Version:  "002",
			Name:     "add_name",
			SQL:      fmt.Sprintf("ALTER TABLE %q ADD COLUMN IF NOT EXISTS name TEXT", dataTable),
			Checksum: "c2",
		},
	}

	// First run
	if err := runner.Up(context.Background(), migrations); err != nil {
		t.Fatalf("first Up: %v", err)
	}

	// Idempotent second run
	if err := runner.Up(context.Background(), migrations); err != nil {
		t.Fatalf("second Up (idempotent): %v", err)
	}

	// Verify table is usable
	if _, err := db.Exec(fmt.Sprintf("INSERT INTO %q (id, name) VALUES ('x', 'test')", dataTable)); err != nil {
		t.Fatalf("insert after migration: %v", err)
	}
}

func TestMigrateIntegChecksumMismatch(t *testing.T) {
	db := pgDB(t)
	migrateTable := uniqueTable("integ_schema_migrations_chk")
	dataTable := uniqueTable("integ_chk_table")

	t.Cleanup(func() {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %q", dataTable))
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %q", migrateTable))
	})

	runner := migrate.Runner{DB: db, Dialect: migrate.DialectPostgres, Table: migrateTable}
	first := []migrate.Migration{
		{
			Version:  "001",
			Name:     "init",
			SQL:      fmt.Sprintf("CREATE TABLE IF NOT EXISTS %q (id TEXT)", dataTable),
			Checksum: "original",
		},
	}
	if err := runner.Up(context.Background(), first); err != nil {
		t.Fatalf("first Up: %v", err)
	}

	tampered := []migrate.Migration{
		{
			Version:  "001",
			Name:     "init",
			SQL:      fmt.Sprintf("CREATE TABLE IF NOT EXISTS %q (id TEXT)", dataTable),
			Checksum: "tampered",
		},
	}
	if err := runner.Up(context.Background(), tampered); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}
