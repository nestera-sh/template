//go:build integration

package trip_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/infra/postgres/transaction"
	"github.com/nestera-sh/template/internal/model"
	repoTrip "github.com/nestera-sh/template/internal/repository/trip"
)

func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set, skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)

	err = pool.Ping(context.Background())
	require.NoError(t, err)

	return pool
}

func TestRepository_Integration(t *testing.T) {
	pool := setupDB(t)
	defer pool.Close()

	repo := repoTrip.NewRepository(pool)
	txMgr := transaction.NewManager(pool)
	ctx := context.Background()

	_, _ = pool.Exec(ctx, "TRUNCATE trips CASCADE")

	t.Run("Save and GetByID", func(t *testing.T) {
		tripID := uuid.New()
		userID := uuid.New()
		driverID := uuid.New()

		tr := &model.Trip{
			ID:             tripID,
			UserID:         userID,
			DriverID:       driverID,
			StartLatitude:  59.9,
			StartLongitude: 30.3,
			EndLatitude:    59.8,
			EndLongitude:   30.4,
			Price:          1500,
			Status:         model.TripStatusActive,
			StartedAt:      time.Now().UTC(),
		}

		err := repo.Save(ctx, tr)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, tripID.String())
		require.NoError(t, err)

		assert.Equal(t, tripID, got.ID)
		assert.Equal(t, int64(1500), got.Price)
	})

	t.Run("driver busy constraint", func(t *testing.T) {
		driverID := uuid.New()

		tr1 := &model.Trip{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			DriverID:  driverID,
			Price:     100,
			Status:    model.TripStatusActive,
			StartedAt: time.Now(),
		}

		err := repo.Save(ctx, tr1)
		require.NoError(t, err)

		tr2 := &model.Trip{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			DriverID:  driverID,
			Price:     100,
			Status:    model.TripStatusActive,
			StartedAt: time.Now(),
		}

		err = repo.Save(ctx, tr2)

		require.ErrorIs(t, err, trip.ErrDriverBusy)
	})

	t.Run("transaction rollback", func(t *testing.T) {
		tripID := uuid.New()

		err := txMgr.Do(ctx, func(ctx context.Context) error {
			tr := &model.Trip{
				ID:        tripID,
				UserID:    uuid.New(),
				DriverID:  uuid.New(),
				Price:     100,
				Status:    model.TripStatusActive,
				StartedAt: time.Now(),
			}

			if err := repo.Save(ctx, tr); err != nil {
				return err
			}

			return trip.ErrTripNotFound
		})

		require.Error(t, err)

		_, err = repo.GetByID(ctx, tripID.String())
		require.ErrorIs(t, err, trip.ErrTripNotFound)
	})

	t.Run("GetStaleTrips selects only active without fresh positions", func(t *testing.T) {
		_, err := pool.Exec(ctx, "TRUNCATE trips CASCADE")
		require.NoError(t, err)

		now := time.Now().UTC()
		staleAfter := 60 * time.Second

		stale := &model.Trip{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			DriverID:       uuid.New(),
			StartLatitude:  59.9,
			StartLongitude: 30.3,
			EndLatitude:    59.8,
			EndLongitude:   30.4,
			Price:          100,
			Status:         model.TripStatusActive,
			StartedAt:      now.Add(-5 * time.Minute),
		}
		require.NoError(t, repo.Save(ctx, stale))

		fresh := &model.Trip{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			DriverID:       uuid.New(),
			StartLatitude:  59.9,
			StartLongitude: 30.3,
			EndLatitude:    59.8,
			EndLongitude:   30.4,
			Price:          100,
			Status:         model.TripStatusActive,
			StartedAt:      now.Add(-5 * time.Minute),
		}
		require.NoError(t, repo.Save(ctx, fresh))

		freshPositionTime := now.Add(-10 * time.Second)

		err = txMgr.Do(ctx, func(ctx context.Context) error {
			return repo.SavePosition(ctx, &model.Position{
				TripID: fresh.ID,
				Coordinates: model.Coordinates{
					Latitude:  59.9,
					Longitude: 30.3,
				},
				RecordedAt: freshPositionTime,
			})
		})
		require.NoError(t, err)

		completed := &model.Trip{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			DriverID:       uuid.New(),
			StartLatitude:  59.9,
			StartLongitude: 30.3,
			EndLatitude:    59.8,
			EndLongitude:   30.4,
			Price:          100,
			Status:         model.TripStatusCompleted,
			StartedAt:      now.Add(-10 * time.Minute),
			FinishedAt:     ptrTime(now.Add(-5 * time.Minute)),
		}
		require.NoError(t, repo.Save(ctx, completed))

		trips, err := repo.GetStaleTrips(ctx, staleAfter)
		require.NoError(t, err)

		require.Len(t, trips, 1)
		assert.Equal(t, stale.ID, trips[0].ID)
	})

	t.Run("SavePosition does not age trip with older position", func(t *testing.T) {
		_, _ = pool.Exec(ctx, "TRUNCATE trips CASCADE")

		now := time.Now().UTC()

		tr := &model.Trip{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			DriverID:       uuid.New(),
			StartLatitude:  59.9,
			StartLongitude: 30.3,
			EndLatitude:    59.8,
			EndLongitude:   30.4,
			Price:          100,
			Status:         model.TripStatusActive,
			StartedAt:      now.Add(-10 * time.Minute),
		}

		require.NoError(t, repo.Save(ctx, tr))

		freshTime := now.Add(-time.Minute)

		err := txMgr.Do(ctx, func(ctx context.Context) error {
			return repo.SavePosition(ctx, &model.Position{
				TripID: tr.ID,
				Coordinates: model.Coordinates{
					Latitude:  59.9,
					Longitude: 30.3,
				},
				RecordedAt: freshTime,
			})
		})

		require.NoError(t, err)

		got, err := repo.GetByID(ctx, tr.ID.String())
		require.NoError(t, err)
		require.NotNil(t, got.LastPositionAt)

		assert.WithinDuration(t, freshTime, *got.LastPositionAt, time.Second)

		// Старая координата не должна откатить last_position_at назад.
		err = txMgr.Do(ctx, func(ctx context.Context) error {
			return repo.SavePosition(ctx, &model.Position{
				TripID: tr.ID,
				Coordinates: model.Coordinates{
					Latitude:  59.8,
					Longitude: 30.4,
				},
				RecordedAt: now.Add(-5 * time.Minute),
			})
		})

		require.NoError(t, err)

		got, err = repo.GetByID(ctx, tr.ID.String())
		require.NoError(t, err)
		require.NotNil(t, got.LastPositionAt)

		assert.WithinDuration(t, freshTime, *got.LastPositionAt, time.Second)
	})
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
