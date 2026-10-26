package get

import (
	"context"

	"github.com/nestera-sh/template/internal/model"
)

type tripRepository interface {
	GetByID(ctx context.Context, id string) (*model.Trip, error)
}
