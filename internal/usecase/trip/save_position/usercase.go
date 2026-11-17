package save_position

import (
	"context"
	"fmt"

	"github.com/nestera-sh/template/internal/model"
)

type UseCase struct {
	repo tripRepository
	tx   txManager
}

func New(repo tripRepository, tx txManager) *UseCase {
	return &UseCase{repo: repo, tx: tx}
}

func (u *UseCase) Execute(ctx context.Context, input Input) error {
	p := &model.Position{
		TripID: input.TripID,
		Coordinates: model.Coordinates{
			Latitude:  input.Latitude,
			Longitude: input.Longitude,
		},
		RecordedAt: input.RecordedAt,
	}

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		return u.repo.SavePosition(ctx, p)
	})
	if err != nil {
		return fmt.Errorf("save position: %w", err)
	}

	return nil
}
