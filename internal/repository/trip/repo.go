package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domainTrip "github.com/nestera-sh/template/internal/domain/trip"
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
		PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build insert: %w", err)
	}
	if _, err := q.Exec(ctx, sql, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domainTrip.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*model.Trip, error) {
	q := r.getQuerier(ctx)
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, domainTrip.ErrTripNotFound
	}
	sql, args, err := squirrel.Select(
		"id", "user_id", "driver_id", "start_latitude", "start_longitude",
		"end_latitude", "end_longitude", "price", "status", "started_at", "finished_at",
		"last_position_at", "created_at", "updated_at",
	).From("trips").Where(squirrel.Eq{"id": uid}).PlaceholderFormat(squirrel.Dollar).ToSql()
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
			return nil, domainTrip.ErrTripNotFound
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}
	return &t, nil
}

func (r *Repository) SaveHistory(ctx context.Context, tripID, fromStatus, toStatus string) error {
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
		return domainTrip.ErrTripNotFound
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
			return domainTrip.ErrTripNotFound
		}
		return domainTrip.ErrTripCompleted
	}
	return nil
}

func (r *Repository) SavePosition(ctx context.Context, p *model.Position) error {
	q := r.getQuerier(ctx)

	sqlInsert, argsInsert, err := squirrel.Insert("trip_positions").
		Columns("trip_id", "latitude", "longitude", "recorded_at").
		Values(p.TripID, p.Coordinates.Latitude, p.Coordinates.Longitude, p.RecordedAt).
		Suffix("RETURNING id, created_at").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build position insert: %w", err)
	}

	if err := q.QueryRow(ctx, sqlInsert, argsInsert...).Scan(&p.ID, &p.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domainTrip.ErrTripNotFound
		}
		return fmt.Errorf("insert position: %w", err)
	}

	sqlUpdate := `
		UPDATE trips
		SET last_position_at = $1, updated_at = now()
		WHERE id = $2
		  AND status = 'active'
		  AND (last_position_at IS NULL OR last_position_at < $1)
	`
	if _, err := q.Exec(ctx, sqlUpdate, p.RecordedAt, p.TripID); err != nil {
		return fmt.Errorf("update last_position_at: %w", err)
	}

	return nil
}

func (r *Repository) GetPositions(ctx context.Context, tripID string) ([]model.Position, error) {
	q := r.getQuerier(ctx)
	uid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, domainTrip.ErrTripNotFound
	}

	sql, args, err := squirrel.Select("id", "trip_id", "latitude", "longitude", "recorded_at", "created_at").
		From("trip_positions").
		Where(squirrel.Eq{"trip_id": uid}).
		OrderBy("recorded_at ASC", "id ASC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build positions select: %w", err)
	}

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query positions: %w", err)
	}
	defer rows.Close()

	var positions []model.Position
	for rows.Next() {
		var p model.Position
		if err := rows.Scan(
			&p.ID, &p.TripID, &p.Coordinates.Latitude, &p.Coordinates.Longitude, &p.RecordedAt, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return positions, nil
}

func (r *Repository) GetStaleTrips(ctx context.Context, staleAfter time.Duration) ([]*model.Trip, error) {
	q := r.getQuerier(ctx)

	sql := `
		SELECT id, driver_id, started_at, last_position_at 
		FROM trips 
		WHERE status = 'active' 
		  AND COALESCE(last_position_at, started_at) < now() - $1::interval
		ORDER BY COALESCE(last_position_at, started_at) ASC
	`
	rows, err := q.Query(ctx, sql, staleAfter)
	if err != nil {
		return nil, fmt.Errorf("query stale trips: %w", err)
	}
	defer rows.Close()

	var trips []*model.Trip
	for rows.Next() {
		var t model.Trip
		if err := rows.Scan(&t.ID, &t.DriverID, &t.StartedAt, &t.LastPositionAt); err != nil {
			return nil, fmt.Errorf("scan stale trip: %w", err)
		}
		trips = append(trips, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return trips, nil
}
