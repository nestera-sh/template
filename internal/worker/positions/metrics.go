package positions

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type OTelMetrics struct {
	staleChecks    metric.Int64Counter
	posRequests    metric.Int64Counter
	posReqDuration metric.Float64Histogram
	queueSize      metric.Int64Gauge
	workersActive  metric.Int64Gauge
}

func NewOTelMetrics() (*OTelMetrics, error) {
	meter := otel.Meter("trip-service")

	staleChecks, err := meter.Int64Counter("tripgo.stale_checks",
		metric.WithDescription("результат обработки поездки сканером"))
	if err != nil {
		return nil, err
	}
	posRequests, err := meter.Int64Counter("tripgo.position_requests",
		metric.WithDescription("результат запроса координаты через Push Service"))
	if err != nil {
		return nil, err
	}
	posReqDuration, err := meter.Float64Histogram("tripgo.position_requests.duration",
		metric.WithUnit("s"),
		metric.WithDescription("длительность вызова Push Service"))
	if err != nil {
		return nil, err
	}
	queueSize, err := meter.Int64Gauge("tripgo.position_requests.queue.size",
		metric.WithDescription("сколько задач в очереди"))
	if err != nil {
		return nil, err
	}
	workersActive, err := meter.Int64Gauge("tripgo.position_requests.workers.active",
		metric.WithDescription("сколько воркеров заняты"))
	if err != nil {
		return nil, err
	}

	return &OTelMetrics{
		staleChecks:    staleChecks,
		posRequests:    posRequests,
		posReqDuration: posReqDuration,
		queueSize:      queueSize,
		workersActive:  workersActive,
	}, nil
}

func (m *OTelMetrics) IncStaleCheck(ctx context.Context, result string) {
	m.staleChecks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *OTelMetrics) IncPositionRequest(ctx context.Context, result string) {
	m.posRequests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *OTelMetrics) IncPositionRequestDuration(ctx context.Context, result string, d time.Duration) {
	m.posReqDuration.Record(ctx, d.Seconds(), metric.WithAttributes(attribute.String("result", result)))
}

func (m *OTelMetrics) SetQueueSize(ctx context.Context, size int64) {
	m.queueSize.Record(ctx, size)
}

func (m *OTelMetrics) SetWorkersActive(ctx context.Context, active int64) {
	m.workersActive.Record(ctx, active)
}

type NoopMetrics struct{}

func (n *NoopMetrics) IncStaleCheck(context.Context, string)                             {}
func (n *NoopMetrics) IncPositionRequest(context.Context, string)                        {}
func (n *NoopMetrics) IncPositionRequestDuration(context.Context, string, time.Duration) {}
func (n *NoopMetrics) SetQueueSize(context.Context, int64)                               {}
func (n *NoopMetrics) SetWorkersActive(context.Context, int64)                           {}
