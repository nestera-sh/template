package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nestera-sh/template/internal/txmanager"
)

type HistoryRepository struct {
	pool *pgxpool.Pool
}

func NewHistoryRepository(pool *pgxpool.Pool) *HistoryRepository {
	return &HistoryRepository{pool: pool}
}

func (r *HistoryRepository) getQuerier(ctx context.Context) interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
} {
	if tx, ok := txmanager.TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *HistoryRepository) Insert(ctx context.Context, tripID uuid.UUID, fromStatus, toStatus string) error {
	q := r.getQuerier(ctx)

	var from any
	if fromStatus != "" {
		from = fromStatus
	}

	sql, args, err := squirrel.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status").
		Values(tripID, from, toStatus).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history: %w", err)
	}

	if _, err := q.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}
