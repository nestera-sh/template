package positions

import (
	"context"
	"log/slog"
	"sync"
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

type Scanner struct {
	repo        repo
	pusher      pusher
	metrics     Metrics
	interval    time.Duration
	staleAfter  time.Duration
	workers     int
	pushTimeout time.Duration

	queue   chan *model.Trip
	mu      sync.Mutex
	inQueue map[string]bool
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
		queue:       make(chan *model.Trip, queueSize),
		inQueue:     make(map[string]bool),
	}
}

func (s *Scanner) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx)
		}()
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	slog.Info("scanner started", "interval", s.interval, "workers", s.workers)

	for {
		select {
		case <-ctx.Done():
			slog.Info("scanner stopping...")
			close(s.queue)
			wg.Wait()
			slog.Info("scanner stopped")
			return
		case <-ticker.C:
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
			if s.metrics != nil {
				s.metrics.IncStaleCheck(ctx, "duplicate")
			}
			continue
		}
		s.inQueue[tripID] = true
		s.mu.Unlock()

		select {
		case s.queue <- t:
			if s.metrics != nil {
				s.metrics.IncStaleCheck(ctx, "queued")
			}
		default:
			s.mu.Lock()
			delete(s.inQueue, tripID)
			s.mu.Unlock()
			slog.Warn("scanner: queue full, dropping trip", "trip_id", tripID)
			if s.metrics != nil {
				s.metrics.IncStaleCheck(ctx, "dropped")
			}
		}
	}
}

func (s *Scanner) worker(ctx context.Context) {
	for t := range s.queue {
		tripID := t.ID.String()

		s.mu.Lock()
		_, exists := s.inQueue[tripID]
		s.mu.Unlock()

		if !exists {
			continue
		}

		if s.metrics != nil {
			s.metrics.SetWorkersActive(ctx, 1)
		}

		start := time.Now()
		pushCtx, cancel := context.WithTimeout(ctx, s.pushTimeout)
		err := s.pusher.RequestPosition(pushCtx, tripID, t.DriverID.String())
		cancel()
		duration := time.Since(start)

		if err != nil {
			slog.Error("worker: push failed", "trip_id", tripID, "err", err)
			if s.metrics != nil {
				s.metrics.IncPositionRequest(ctx, "error")
				s.metrics.IncPositionRequestDuration(ctx, "error", duration)
			}
		} else {
			slog.Info("worker: position requested", "trip_id", tripID)
			if s.metrics != nil {
				s.metrics.IncPositionRequest(ctx, "success")
				s.metrics.IncPositionRequestDuration(ctx, "success", duration)
			}
		}

		if s.metrics != nil {
			s.metrics.SetWorkersActive(ctx, -1)
		}

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
