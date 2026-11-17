package get_positions

import (
	"context"

	"github.com/nestera-sh/template/internal/model"
)

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=get_positions_test

type tripRepository interface {
	GetPositions(ctx context.Context, tripID string) ([]model.Position, error)
}
