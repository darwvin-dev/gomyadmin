// Package gormstore adapts a GORM connection to GoMyAdmin's AdminStore.
package gormstore

import (
	"github.com/darwvin-dev/gomyadmin/pkg/adapters/sqlstore"
	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"gorm.io/gorm"
)

// New returns a database/sql-backed store using the *sql.DB managed by GORM.
// The application keeps ownership of the GORM connection and its lifecycle.
func New(db *gorm.DB, app *admin.App, dialect sqlstore.Dialect) (*sqlstore.Store, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	return sqlstore.New(sqlDB, app, dialect), nil
}

func MySQL(db *gorm.DB, app *admin.App) (*sqlstore.Store, error) {
	return New(db, app, sqlstore.DialectMySQL)
}

func SQLite(db *gorm.DB, app *admin.App) (*sqlstore.Store, error) {
	return New(db, app, sqlstore.DialectSQLite)
}
