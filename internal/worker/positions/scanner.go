package positions

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nestera-sh/template/internal/model"
)

type repo interface {
	GetStaleTrips(ctx context.Context, staleAfter time.Duration) ([]*model.Trip, error)
}

type pusher interface {
	RequestPosition(ctx context.Context, tripID, driverID string) error
}

type Metrics interface {
	IncStaleCheck(ctx context.Context, result string)
	IncPositionRequest(ctx context.Context, result string)
	IncPositionRequestDuration(ctx context.Context, result string, duration time.Duration)
	SetQueueSize(ctx context.Context, size int64)
	SetWorkersActive(ctx context.Context, active int64)
}

type tickerFactory func(d time.Duration) <-chan time.Time

type Scanner struct {
	repo        repo
	pusher      pusher
	metrics     Metrics
	interval    time.Duration
	staleAfter  time.Duration
	workers     int
	pushTimeout time.Duration
	newTicker   tickerFactory

	queue   chan *model.Trip
	mu      sync.Mutex
	inQueue map[string]bool

	activeWorkers atomic.Int64

	wg sync.WaitGroup
}

func NewScanner(
	repo repo,
	pusher pusher,
	metrics Metrics,
	interval time.Duration,
	staleAfter time.Duration,
	workers int,
	queueSize int,
	pushTimeout time.Duration,
) *Scanner {
	return &Scanner{
		repo:        repo,
		pusher:      pusher,
		metrics:     metrics,
		interval:    interval,
		staleAfter:  staleAfter,
		workers:     workers,
		pushTimeout: pushTimeout,
		newTicker: func(d time.Duration) <-chan time.Time {
			return time.NewTicker(d).C
		},
		queue:   make(chan *model.Trip, queueSize),
		inQueue: make(map[string]bool),
	}
}

func (s *Scanner) WithTicker(tf tickerFactory) *Scanner {
	s.newTicker = tf
	return s
}

func (s *Scanner) Start(ctx context.Context, shutdownTimeout time.Duration) {
	s.wg.Add(s.workers)
	for i := 0; i < s.workers; i++ {
		go func() {
			defer s.wg.Done()
			s.worker(ctx)
		}()
	}

	slog.Info("scanner started", "interval", s.interval, "workers", s.workers)

	s.run(ctx)

	slog.Info("scanner: stopping workers, draining queue")
	close(s.queue)

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("scanner stopped")
	case <-time.After(shutdownTimeout):
		slog.Warn("scanner: shutdown timeout exceeded, some workers still running")
	}
}

func (s *Scanner) run(ctx context.Context) {
	tickCh := s.newTicker(s.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tickCh:
			s.scan(ctx)
		}
	}
}

func (s *Scanner) scan(ctx context.Context) {
	trips, err := s.repo.GetStaleTrips(ctx, s.staleAfter)
	if err != nil {
		slog.Error("scanner: get stale trips", "err", err)
		return
	}

	for _, t := range trips {
		tripID := t.ID.String()

		s.mu.Lock()
		if s.inQueue[tripID] {
			s.mu.Unlock()
			s.metrics.IncStaleCheck(ctx, "duplicate")
			continue
		}
		s.inQueue[tripID] = true
		s.mu.Unlock()

		select {
		case s.queue <- t:
			s.metrics.IncStaleCheck(ctx, "queued")
			s.metrics.SetQueueSize(ctx, int64(len(s.queue)))
		default:
			s.mu.Lock()
			delete(s.inQueue, tripID)
			s.mu.Unlock()
			slog.Warn("scanner: queue full, dropping trip", "trip_id", tripID)
			s.metrics.IncStaleCheck(ctx, "dropped")
		}
	}
}

func (s *Scanner) worker(ctx context.Context) {
	for t := range s.queue {
		s.metrics.SetQueueSize(context.Background(), int64(len(s.queue)))

		tripID := t.ID.String()

		s.mu.Lock()
		_, exists := s.inQueue[tripID]
		s.mu.Unlock()
		if !exists {
			continue
		}

		active := s.activeWorkers.Add(1)
		s.metrics.SetWorkersActive(context.Background(), active)

		start := time.Now()
		pushCtx, cancel := context.WithTimeout(ctx, s.pushTimeout)
		err := s.pusher.RequestPosition(pushCtx, tripID, t.DriverID.String())
		cancel()
		duration := time.Since(start)

		metricCtx := context.Background()
		if err != nil {
			slog.Warn("worker: push failed", "trip_id", tripID, "err", err)
			s.metrics.IncPositionRequest(metricCtx, "error")
			s.metrics.IncPositionRequestDuration(metricCtx, "error", duration)
		} else {
			slog.Info("worker: position requested", "trip_id", tripID)
			s.metrics.IncPositionRequest(metricCtx, "success")
			s.metrics.IncPositionRequestDuration(metricCtx, "success", duration)
		}

		active = s.activeWorkers.Add(-1)
		s.metrics.SetWorkersActive(metricCtx, active)

		s.mu.Lock()
		delete(s.inQueue, tripID)
		s.mu.Unlock()
	}
}

func (s *Scanner) CancelTask(tripID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inQueue, tripID)
}
