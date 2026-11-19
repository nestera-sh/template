package save_position

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/nestera-sh/template/internal/model"
)

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=save_position_test

type (
	tripRepository interface {
		GetByID(ctx context.Context, id string) (*model.Trip, error)
		SavePosition(ctx context.Context, p *model.Position) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metrics interface {
		IncPositionsSaved(ctx context.Context)
	}
)

type Input struct {
	TripID     uuid.UUID
	Latitude   float64
	Longitude  float64
	RecordedAt time.Time
}
