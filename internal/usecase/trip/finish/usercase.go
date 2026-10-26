package finish

import (
	"context"
	"fmt"
	"time"

	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/model"
)

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

func (u *UseCase) Execute(ctx context.Context, id string) (*model.Trip, error) {
	var finishedTrip *model.Trip

	err := u.txMgr.Do(ctx, func(ctx context.Context) error {
		t, err := u.tripRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}

		if err := trip.Finish(t, time.Now().UTC()); err != nil {
			return err
		}

		if err := u.tripRepo.Finish(ctx, t.ID.String()); err != nil {
			return err
		}

		if err := u.tripRepo.SaveHistory(ctx, t.ID.String(), string(model.TripStatusActive), string(model.TripStatusCompleted)); err != nil {
			return err
		}

		finishedTrip = t
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("finish trip: %w", err)
	}

	return finishedTrip, nil
}
