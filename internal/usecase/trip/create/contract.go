package create

import (
	"context"

	"github.com/nestera-sh/template/internal/model"
)

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=create_test

type (
	tripRepository interface {
		Save(ctx context.Context, t *model.Trip) error
		SaveHistory(ctx context.Context, tripID, fromStatus, toStatus string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metrics interface {
		IncTripsCreated(ctx context.Context)
	}
)
