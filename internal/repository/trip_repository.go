package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nestera-sh/template/internal/model"
	"github.com/nestera-sh/template/internal/txmanager"
)

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type TripRepository struct {
	pool *pgxpool.Pool
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{pool: pool}
}

func (r *TripRepository) getQuerier(ctx context.Context) querier {
	if tx, ok := txmanager.TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *TripRepository) Create(ctx context.Context, trip *model.Trip) error {
	q := r.getQuerier(ctx)

	sql, args, err := squirrel.Insert("trips").
		Columns("id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at").
		Values(trip.ID, trip.UserID, trip.DriverID,
			trip.StartLatitude, trip.StartLongitude,
			trip.EndLatitude, trip.EndLongitude,
			trip.Price, trip.Status, trip.StartedAt).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert: %w", err)
	}

	if _, err := q.Exec(ctx, sql, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	q := r.getQuerier(ctx)

	sql, args, err := squirrel.Select(
		"id", "user_id", "driver_id",
		"start_latitude", "start_longitude",
		"end_latitude", "end_longitude",
		"price", "status", "started_at", "finished_at",
		"last_position_at", "created_at", "updated_at").
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select: %w", err)
	}

	var t model.Trip
	err = q.QueryRow(ctx, sql, args...).Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartLatitude, &t.StartLongitude,
		&t.EndLatitude, &t.EndLongitude,
		&t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
		&t.LastPositionAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTripNotFound
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}
	return &t, nil
}

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	q := r.getQuerier(ctx)

	sql, args, err := squirrel.Update("trips").
		Set("status", model.TripStatusCompleted).
		Set("finished_at", squirrel.Expr("now()")).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id, "status": model.TripStatusActive}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update: %w", err)
	}

	tag, err := q.Exec(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("update trip: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return nil, r.classifyFinishFailure(ctx, id)
	}
	return r.GetByID(ctx, id)
}

func (r *TripRepository) classifyFinishFailure(ctx context.Context, id uuid.UUID) error {
	q := r.getQuerier(ctx)

	sql, args, err := squirrel.Select("status").
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build select status: %w", err)
	}

	var status string
	err = q.QueryRow(ctx, sql, args...).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrTripNotFound
		}
		return fmt.Errorf("scan status: %w", err)
	}
	return model.ErrTripCompleted
}
