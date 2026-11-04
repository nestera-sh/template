package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(
	ctx context.Context,
	dsn string,
	maxConnections int,
	minConnections int,
	connectTimeout time.Duration,
	maxConnLifetime time.Duration,
) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	cfg.MaxConns = int32(maxConnections) //nolint:gosec
	cfg.MinConns = int32(minConnections) //nolint:gosec // переполнения не будет, потому что размер пула не модет превысить int32
	cfg.MaxConnLifetime = maxConnLifetime
	cfg.ConnConfig.ConnectTimeout = connectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}
