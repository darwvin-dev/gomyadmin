package introspect

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Schema struct {
	Tables []Table `json:"tables"`
}

type Table struct {
	Schema     string   `json:"schema"`
	Name       string   `json:"name"`
	Columns    []Column `json:"columns"`
	PrimaryKey []string `json:"primary_key"`
}

type Column struct {
	Name       string `json:"name"`
	DataType   string `json:"data_type"`
	Nullable   bool   `json:"nullable"`
	Default    string `json:"default,omitempty"`
	IsIdentity bool   `json:"is_identity"`
}

func Load(ctx context.Context, databaseURL string) (Schema, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return Schema{}, err
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
select table_schema, table_name
from information_schema.tables
where table_type = 'BASE TABLE'
  and table_schema not in ('pg_catalog', 'information_schema')
order by table_schema, table_name`)
	if err != nil {
		return Schema{}, err
	}
	defer rows.Close()

	var schema Schema
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Schema, &table.Name); err != nil {
			return Schema{}, err
		}
		columns, err := loadColumns(ctx, pool, table.Schema, table.Name)
		if err != nil {
			return Schema{}, err
		}
		table.Columns = columns
		table.PrimaryKey, err = loadPrimaryKey(ctx, pool, table.Schema, table.Name)
		if err != nil {
			return Schema{}, err
		}
		schema.Tables = append(schema.Tables, table)
	}
	return schema, rows.Err()
}

func loadColumns(ctx context.Context, pool *pgxpool.Pool, schema, table string) ([]Column, error) {
	rows, err := pool.Query(ctx, `
select column_name, data_type, is_nullable = 'YES', coalesce(column_default, ''), is_identity = 'YES'
from information_schema.columns
where table_schema = $1 and table_name = $2
order by ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var column Column
		if err := rows.Scan(&column.Name, &column.DataType, &column.Nullable, &column.Default, &column.IsIdentity); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, rows.Err()
}

func loadPrimaryKey(ctx context.Context, pool *pgxpool.Pool, schema, table string) ([]string, error) {
	rows, err := pool.Query(ctx, `
select kcu.column_name
from information_schema.table_constraints tc
join information_schema.key_column_usage kcu
  on tc.constraint_name = kcu.constraint_name
 and tc.table_schema = kcu.table_schema
where tc.constraint_type = 'PRIMARY KEY'
  and tc.table_schema = $1
  and tc.table_name = $2
order by kcu.ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, rows.Err()
}
