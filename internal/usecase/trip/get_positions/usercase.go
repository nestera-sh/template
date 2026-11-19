package get_positions

import (
	"context"
	"fmt"

	"github.com/nestera-sh/template/internal/model"
)

type UseCase struct {
	repo tripRepository
}

func New(repo tripRepository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Execute(ctx context.Context, tripID string) ([]model.Position, error) {
	pos, err := u.repo.GetPositions(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get positions: %w", err)
	}
	return pos, nil
}
