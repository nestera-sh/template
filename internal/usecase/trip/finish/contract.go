package finish

import (
	"context"

	"github.com/nestera-sh/template/internal/model"
)

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=finish_test

type (
	tripRepository interface {
		GetByID(ctx context.Context, id string) (*model.Trip, error)
		SaveHistory(ctx context.Context, tripID, fromStatus, toStatus string) error
		Finish(ctx context.Context, id string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metrics interface {
		IncTripsCompleted(ctx context.Context)
	}
)
