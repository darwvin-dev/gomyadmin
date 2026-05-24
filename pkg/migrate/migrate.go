// Package migrate provides a small production-oriented SQL migration runner.
package migrate

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectMySQL    Dialect = "mysql"
	DialectSQLite   Dialect = "sqlite"
)

type Migration struct {
	Version  string
	Name     string
	SQL      string
	Checksum string
}

type Runner struct {
	DB      *sql.DB
	Dialect Dialect
	Table   string
	Now     func() time.Time
}

func FromFS(files fs.FS, root string) ([]Migration, error) {
	entries, err := fs.ReadDir(files, root)
	if err != nil {
		return nil, err
	}
	migrations := []Migration{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		body, err := fs.ReadFile(files, filepath.ToSlash(filepath.Join(root, entry.Name())))
		if err != nil {
			return nil, err
		}
		version, name, _ := strings.Cut(strings.TrimSuffix(entry.Name(), ".sql"), "_")
		sqlText := string(body)
		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			SQL:      sqlText,
			Checksum: checksum(sqlText),
		})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

func (r Runner) Up(ctx context.Context, migrations []Migration) error {
	if r.DB == nil {
		return fmt.Errorf("migrate.Runner: DB is required")
	}
	if err := r.ensureTable(ctx); err != nil {
		return err
	}
	applied, err := r.applied(ctx)
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		if previous, ok := applied[migration.Version]; ok {
			if previous != migration.Checksum {
				return fmt.Errorf("migration %s checksum mismatch", migration.Version)
			}
			continue
		}
		if err := r.apply(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}

func (r Runner) ensureTable(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, `create table if not exists `+r.table()+` (
version varchar(128) primary key,
name varchar(255) not null,
checksum varchar(64) not null,
applied_at timestamp not null
)`)
	return err
}

func (r Runner) applied(ctx context.Context) (map[string]string, error) {
	rows, err := r.DB.QueryContext(ctx, `select version, checksum from `+r.table())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, err
		}
		result[version] = checksum
	}
	return result, rows.Err()
}

func (r Runner) apply(ctx context.Context, migration Migration) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("migration %s failed: %w", migration.Version, err)
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	if _, err = tx.ExecContext(ctx, `insert into `+r.table()+` (version, name, checksum, applied_at) values (`+r.placeholders(4)+`)`, migration.Version, migration.Name, migration.Checksum, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (r Runner) table() string {
	if r.Table != "" {
		return quote(r.Table, r.Dialect)
	}
	return quote("gomyadmin_schema_migrations", r.Dialect)
}

func (r Runner) placeholders(n int) string {
	out := make([]string, n)
	for i := range out {
		if r.Dialect == DialectPostgres {
			out[i] = fmt.Sprintf("$%d", i+1)
		} else {
			out[i] = "?"
		}
	}
	return strings.Join(out, ", ")
}

func quote(identifier string, dialect Dialect) string {
	if dialect == DialectMySQL {
		return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func checksum(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
