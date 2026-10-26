package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/infra/postgres/transaction"
	"github.com/nestera-sh/template/internal/model"
)

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) getQuerier(ctx context.Context) querier {
	if tx, ok := transaction.FromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *Repository) Save(ctx context.Context, t *model.Trip) error {
	q := r.getQuerier(ctx)

	sql, args, err := squirrel.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").
		Values(t.ID, t.UserID, t.DriverID, t.StartLatitude, t.StartLongitude, t.EndLatitude, t.EndLongitude, t.Price, t.Status, t.StartedAt, t.FinishedAt).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert: %w", err)
	}

	if _, err := q.Exec(ctx, sql, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return trip.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*model.Trip, error) {
	q := r.getQuerier(ctx)
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, trip.ErrTripNotFound
	}

	sql, args, err := squirrel.Select(
		"id", "user_id", "driver_id", "start_latitude", "start_longitude",
		"end_latitude", "end_longitude", "price", "status", "started_at", "finished_at",
		"last_position_at", "created_at", "updated_at").
		From("trips").Where(squirrel.Eq{"id": uid}).PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select: %w", err)
	}

	var t model.Trip
	err = q.QueryRow(ctx, sql, args...).Scan(
		&t.ID, &t.UserID, &t.DriverID, &t.StartLatitude, &t.StartLongitude,
		&t.EndLatitude, &t.EndLongitude, &t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
		&t.LastPositionAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, trip.ErrTripNotFound
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}
	return &t, nil
}

func (r *Repository) SaveHistory(ctx context.Context, tripID string, fromStatus, toStatus string) error {
	q := r.getQuerier(ctx)

	var from any
	if fromStatus != "" {
		from = fromStatus
	}

	sql, args, err := squirrel.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status").
		Values(tripID, from, toStatus).
		PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build history insert: %w", err)
	}

	if _, err := q.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}

func (r *Repository) Finish(ctx context.Context, id string) error {
	q := r.getQuerier(ctx)
	uid, err := uuid.Parse(id)
	if err != nil {
		return trip.ErrTripNotFound
	}

	sql, args, err := squirrel.Update("trips").
		Set("status", model.TripStatusCompleted).
		Set("finished_at", squirrel.Expr("now()")).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": uid, "status": model.TripStatusActive}).
		PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return err
	}

	tag, err := q.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		var status string
		err := q.QueryRow(ctx, "SELECT status FROM trips WHERE id = $1", uid).Scan(&status)
		if err != nil {
			return trip.ErrTripNotFound
		}
		return trip.ErrTripCompleted
	}
	return nil
}
