package gormstore

import (
	"path/filepath"
	"testing"

	glebarez "github.com/glebarez/sqlite"
	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/adapters/sqlstore"
	"gorm.io/gorm"
)

type item struct{}

func openGorm(t *testing.T) *gorm.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gorm.db")
	db, err := gorm.Open(glebarez.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestNewReturnsStore(t *testing.T) {
	db := openGorm(t)
	app := admin.New("Test")
	app.Resource(item{}).TableName("items").Field("ID").String().Primary()

	store, err := New(db, app, sqlstore.DialectSQLite)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestMySQLReturnsStore(t *testing.T) {
	db := openGorm(t)
	app := admin.New("Test")
	app.Resource(item{}).TableName("items").Field("ID").String().Primary()

	store, err := MySQL(db, app)
	if err != nil {
		t.Fatalf("MySQL: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestSQLiteReturnsStore(t *testing.T) {
	db := openGorm(t)
	app := admin.New("Test")
	app.Resource(item{}).TableName("items").Field("ID").String().Primary()

	store, err := SQLite(db, app)
	if err != nil {
		t.Fatalf("SQLite: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}
