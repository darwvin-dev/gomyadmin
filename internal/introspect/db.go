package introspect

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// querier is the minimal interface needed to run queries.
// *pgxpool.Pool satisfies it; tests can supply a fake.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Load connects to a PostgreSQL database and introspects all user tables,
// returning a Schema containing their columns and primary-key information.
func Load(ctx context.Context, databaseURL string) (Schema, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return Schema{}, err
	}
	defer pool.Close()
	return loadSchema(ctx, pool)
}

// loadSchema does the actual introspection work against any querier.
func loadSchema(ctx context.Context, q querier) (Schema, error) {
	rows, err := q.Query(ctx, `
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
		columns, err := loadColumns(ctx, q, table.Schema, table.Name)
		if err != nil {
			return Schema{}, err
		}
		table.Columns = columns
		table.PrimaryKey, err = loadPrimaryKey(ctx, q, table.Schema, table.Name)
		if err != nil {
			return Schema{}, err
		}
		schema.Tables = append(schema.Tables, table)
	}
	return schema, rows.Err()
}

func loadColumns(ctx context.Context, q querier, schema, table string) ([]Column, error) {
	rows, err := q.Query(ctx, `
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

func loadPrimaryKey(ctx context.Context, q querier, schema, table string) ([]string, error) {
	rows, err := q.Query(ctx, `
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
