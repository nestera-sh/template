package create

import (
	"context"

	"github.com/nestera-sh/template/internal/model"
)

type (
	tripRepository interface {
		Save(ctx context.Context, t *model.Trip) error
		SaveHistory(ctx context.Context, tripID string, fromStatus, toStatus string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)
