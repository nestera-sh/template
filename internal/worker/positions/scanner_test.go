package positions_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/nestera-sh/template/internal/model"
	"github.com/nestera-sh/template/internal/worker/positions"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type fakeRepo struct {
	trips []*model.Trip
	err   error

	mu sync.Mutex
}

func (f *fakeRepo) GetStaleTrips(
	_ context.Context,
	_ time.Duration,
) ([]*model.Trip, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.trips, f.err
}

type fakePusher struct {
	mu sync.Mutex

	calls   []string
	err     error
	block   chan struct{}
	started chan struct{}
}

func (f *fakePusher) RequestPosition(
	ctx context.Context,
	tripID string,
	_ string,
) error {
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}

	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	f.mu.Lock()
	f.calls = append(f.calls, tripID)
	f.mu.Unlock()

	return f.err
}

func (f *fakePusher) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

func makeTrip() *model.Trip {
	return &model.Trip{
		ID:       uuid.New(),
		DriverID: uuid.New(),
		Status:   model.TripStatusActive,
	}
}

func oneTick(_ time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func receive(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestScanner_DedupAndQueue(t *testing.T) {
	trip := makeTrip()

	repo := &fakeRepo{
		trips: []*model.Trip{trip, trip},
	}

	block := make(chan struct{})
	pusher := &fakePusher{
		block:   block,
		started: make(chan struct{}, 1),
	}

	var queued atomic.Int64
	var duplicate atomic.Int64

	metrics := &countingMetrics{
		queued:       &queued,
		duplicate:    &duplicate,
		staleResults: make(chan string, 10),
	}

	s := positions.NewScanner(
		repo,
		pusher,
		metrics,
		time.Second,
		time.Minute,
		1,
		10,
		time.Second,
	).WithTicker(oneTick)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		defer close(done)
		s.Start(ctx, time.Second)
	}()

	receive(t, pusher.started)

	firstResult := <-metrics.staleResults
	secondResult := <-metrics.staleResults

	assert.Equal(t, "queued", firstResult)
	assert.Equal(t, "duplicate", secondResult)

	assert.Equal(t, int64(1), queued.Load())
	assert.Equal(t, int64(1), duplicate.Load())

	close(block)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scanner did not stop")
	}

	assert.Equal(t, 1, pusher.callCount())
}

func TestScanner_QueueFullDrops(t *testing.T) {
	trips := []*model.Trip{
		makeTrip(),
		makeTrip(),
		makeTrip(),
	}

	repo := &fakeRepo{
		trips: trips,
	}

	block := make(chan struct{})

	pusher := &fakePusher{
		block:   block,
		started: make(chan struct{}, 1),
	}

	var dropped atomic.Int64

	metrics := &countingMetrics{
		dropped:      &dropped,
		staleResults: make(chan string, 10),
	}

	s := positions.NewScanner(
		repo,
		pusher,
		metrics,
		time.Second,
		time.Minute,
		1,
		1,
		time.Second,
	).WithTicker(oneTick)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		defer close(done)
		s.Start(ctx, time.Second)
	}()

	receive(t, pusher.started)

	foundDrop := false

	for i := 0; i < len(trips); i++ {
		select {
		case result := <-metrics.staleResults:
			if result == "dropped" {
				foundDrop = true
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for queue result")
		}
	}

	assert.True(t, foundDrop)
	assert.GreaterOrEqual(t, dropped.Load(), int64(1))

	close(block)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scanner did not stop")
	}
}

func TestScanner_CancelTaskSkipsPush(t *testing.T) {
	firstTrip := makeTrip()
	secondTrip := makeTrip()

	repo := &fakeRepo{
		trips: []*model.Trip{
			firstTrip,
			secondTrip,
		},
	}

	block := make(chan struct{})

	pusher := &fakePusher{
		block:   block,
		started: make(chan struct{}, 1),
	}

	metrics := &countingMetrics{
		staleResults: make(chan string, 10),
	}

	s := positions.NewScanner(
		repo,
		pusher,
		metrics,
		time.Second,
		time.Minute,
		1,
		10,
		time.Second,
	).WithTicker(oneTick)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		defer close(done)
		s.Start(ctx, time.Second)
	}()

	receive(t, pusher.started)

	for i := 0; i < 2; i++ {
		select {
		case result := <-metrics.staleResults:
			assert.Equal(t, "queued", result)
		case <-time.After(time.Second):
			t.Fatal("scanner did not enqueue both trips")
		}
	}

	s.CancelTask(secondTrip.ID.String())

	close(block)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scanner did not stop")
	}

	assert.Equal(t, 1, pusher.callCount())
}

func TestScanner_WorkerErrorDoesNotCrash(t *testing.T) {
	repo := &fakeRepo{
		trips: []*model.Trip{
			makeTrip(),
			makeTrip(),
		},
	}

	pusher := &fakePusher{
		err: errors.New("push service unavailable"),
	}

	var errorCount atomic.Int64

	metrics := &countingMetrics{
		errorCount: &errorCount,
	}

	s := positions.NewScanner(
		repo,
		pusher,
		metrics,
		time.Second,
		time.Minute,
		1,
		10,
		time.Second,
	).WithTicker(oneTick)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		defer close(done)
		s.Start(ctx, time.Second)
	}()

	require.Eventually(t, func() bool {
		return pusher.callCount() == 2
	}, time.Second, time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scanner did not stop")
	}

	assert.Equal(t, int64(2), errorCount.Load())
}

func TestScanner_ContextCancelStopsWorkers(t *testing.T) {
	repo := &fakeRepo{}

	s := positions.NewScanner(
		repo,
		&fakePusher{},
		&countingMetrics{},
		time.Hour,
		time.Minute,
		4,
		10,
		time.Second,
	)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		defer close(done)
		s.Start(ctx, time.Second)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scanner did not stop")
	}
}

type countingMetrics struct {
	queued       *atomic.Int64
	duplicate    *atomic.Int64
	dropped      *atomic.Int64
	errorCount   *atomic.Int64
	staleResults chan string
}

func (m *countingMetrics) inc(counter *atomic.Int64) {
	if counter != nil {
		counter.Add(1)
	}
}

func (m *countingMetrics) IncStaleCheck(
	_ context.Context,
	result string,
) {
	switch result {
	case "queued":
		m.inc(m.queued)
	case "duplicate":
		m.inc(m.duplicate)
	case "dropped":
		m.inc(m.dropped)
	}

	if m.staleResults != nil {
		m.staleResults <- result
	}
}

func (m *countingMetrics) IncPositionRequest(
	_ context.Context,
	result string,
) {
	if result == "error" {
		m.inc(m.errorCount)
	}
}

func (m *countingMetrics) IncPositionRequestDuration(
	context.Context,
	string,
	time.Duration,
) {
}

func (m *countingMetrics) SetQueueSize(context.Context, int64) {
}

func (m *countingMetrics) SetWorkersActive(context.Context, int64) {
}
