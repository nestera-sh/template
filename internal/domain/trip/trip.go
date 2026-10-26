package trip

import (
	"time"

	"github.com/nestera-sh/template/internal/model"
)

func Finish(t *model.Trip, at time.Time) error {
	if t.Status != model.TripStatusActive {
		return ErrTripCompleted
	}

	t.Status = model.TripStatusCompleted
	t.FinishedAt = &at
	return nil
}
