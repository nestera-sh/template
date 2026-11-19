package save_position

import (
	"context"
	"fmt"

	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/model"
)

type UseCase struct {
	repo    tripRepository
	tx      txManager
	metrics metrics
}

func New(repo tripRepository, tx txManager, metrics metrics) *UseCase {
	return &UseCase{repo: repo, tx: tx, metrics: metrics}
}

func (u *UseCase) Execute(ctx context.Context, input Input) (*model.Position, error) {
	var saved *model.Position

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		t, err := u.repo.GetByID(ctx, input.TripID.String())
		if err != nil {
			return err
		}
		if t.Status != model.TripStatusActive {
			return trip.ErrTripCompleted
		}

		p := &model.Position{
			TripID: input.TripID,
			Coordinates: model.Coordinates{
				Latitude:  input.Latitude,
				Longitude: input.Longitude,
			},
			RecordedAt: input.RecordedAt,
		}
		if err := u.repo.SavePosition(ctx, p); err != nil {
			return err
		}
		saved = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("save position: %w", err)
	}

	if u.metrics != nil {
		u.metrics.IncPositionsSaved(ctx)
	}

	return saved, nil
}
