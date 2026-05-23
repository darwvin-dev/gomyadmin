package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DatabaseURL      string
	MaxConns         int32
	MinConns         int32
	ConnectTimeout   time.Duration
	StatementTimeout time.Duration
}

func Connect(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if config.MaxConns > 0 {
		poolConfig.MaxConns = config.MaxConns
	}
	if config.MinConns > 0 {
		poolConfig.MinConns = config.MinConns
	}
	if config.ConnectTimeout > 0 {
		poolConfig.ConnConfig.ConnectTimeout = config.ConnectTimeout
	}
	if config.StatementTimeout > 0 {
		timeout := config.StatementTimeout
		poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, "select set_config('statement_timeout', $1, false)", timeout.String())
			return err
		}
	}
	return pgxpool.NewWithConfig(ctx, poolConfig)
}
