package create

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nestera-sh/template/internal/model"
)

type CreateTripInput struct {
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
}

type UseCase struct {
	tripRepo tripRepository
	txMgr    txManager
}

func New(tripRepo tripRepository, txMgr txManager) *UseCase {
	return &UseCase{
		tripRepo: tripRepo,
		txMgr:    txMgr,
	}
}

func (u *UseCase) Execute(ctx context.Context, input CreateTripInput) (*model.Trip, error) {
	t := &model.Trip{
		ID:             uuid.New(),
		UserID:         input.UserID,
		DriverID:       input.DriverID,
		StartLatitude:  input.StartLatitude,
		StartLongitude: input.StartLongitude,
		EndLatitude:    input.EndLatitude,
		EndLongitude:   input.EndLongitude,
		Price:          input.Price,
		Status:         model.TripStatusActive,
		StartedAt:      time.Now().UTC(),
	}

	err := u.txMgr.Do(ctx, func(ctx context.Context) error {
		if err := u.tripRepo.Save(ctx, t); err != nil {
			return err
		}
		return u.tripRepo.SaveHistory(ctx, t.ID.String(), "", string(model.TripStatusActive))
	})

	if err != nil {
		return nil, fmt.Errorf("create trip: %w", err)
	}

	return t, nil
}
