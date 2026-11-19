package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	tripsCreated   metric.Int64Counter
	tripsCompleted metric.Int64Counter
	tripsActive    metric.Int64UpDownCounter
	positionsSaved metric.Int64Counter
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
	active, err := meter.Int64UpDownCounter("tripgo.trips.active")
	if err != nil {
		return nil, err
	}
	positionsSaved, err := meter.Int64Counter("tripgo.positions.saved")
	if err != nil {
		return nil, err
	}

	return &Metrics{
		tripsCreated:   created,
		tripsCompleted: completed,
		tripsActive:    active,
		positionsSaved: positionsSaved,
	}, nil
}

func (m *Metrics) IncTripsCreated(ctx context.Context) {
	m.tripsCreated.Add(ctx, 1)
	m.tripsActive.Add(ctx, 1)
}

func (m *Metrics) IncTripsCompleted(ctx context.Context) {
	m.tripsCompleted.Add(ctx, 1)
	m.tripsActive.Add(ctx, -1)
}

func (m *Metrics) IncPositionsSaved(ctx context.Context) {
	m.positionsSaved.Add(ctx, 1)
}
