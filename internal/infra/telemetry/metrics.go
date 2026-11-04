package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	tripsCreated   metric.Int64Counter
	tripsCompleted metric.Int64Counter
}

func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("trip-service")

	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		return nil, err
	}

	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		return nil, err
	}

	return &Metrics{
		tripsCreated:   created,
		tripsCompleted: completed,
	}, nil
}

func (m *Metrics) IncTripsCreated(ctx context.Context) {
	m.tripsCreated.Add(ctx, 1)
}

func (m *Metrics) IncTripsCompleted(ctx context.Context) {
	m.tripsCompleted.Add(ctx, 1)
}
