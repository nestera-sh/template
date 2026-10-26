package get

import (
	"context"
	"fmt"

	"github.com/nestera-sh/template/internal/model"
)

type UseCase struct {
	tripRepo tripRepository
}

func New(tripRepo tripRepository) *UseCase {
	return &UseCase{tripRepo: tripRepo}
}

func (u *UseCase) Execute(ctx context.Context, id string) (*model.Trip, error) {
	t, err := u.tripRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}
	return t, nil
}
