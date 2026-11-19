package positions

import (
	"context"
	"time"
)

type NoopMetrics struct{}

func (n *NoopMetrics) IncStaleCheck(ctx context.Context, result string)      {}
func (n *NoopMetrics) IncPositionRequest(ctx context.Context, result string) {}
func (n *NoopMetrics) IncPositionRequestDuration(ctx context.Context, result string, duration time.Duration) {
}
func (n *NoopMetrics) SetQueueSize(ctx context.Context, size int64)       {}
func (n *NoopMetrics) SetWorkersActive(ctx context.Context, active int64) {}
