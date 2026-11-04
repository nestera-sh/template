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
}
