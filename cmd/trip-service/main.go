package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nestera-sh/template/internal/config"
	"github.com/nestera-sh/template/internal/db"
	api "github.com/nestera-sh/template/internal/generated"
	"github.com/nestera-sh/template/internal/handler"
	"github.com/nestera-sh/template/internal/repository"
	"github.com/nestera-sh/template/internal/txmanager"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	pool, err := db.New(
		context.Background(),
		cfg.DatabaseURL,
		cfg.DatabaseMaxConns,
		cfg.DatabaseMinConns,
		cfg.DatabaseConnectTimeout,
		cfg.DatabaseMaxConnLifetime,
	)
	if err != nil {
		slog.Error("connect to db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	tripRepo := repository.NewTripRepository(pool)
	historyRepo := repository.NewHistoryRepository(pool)
	txManager := txmanager.New(pool)
	tripHandler := handler.NewTripHandler(tripRepo, historyRepo, txManager, cfg.DatabaseQueryTimeout)

	mux := chi.NewRouter()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	api.HandlerWithOptions(&tripServer{handler: tripHandler, pool: pool}, api.ChiServerOptions{
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			handler.RespondInvalidParam(w, r, err)
		},
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("starting server", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		_ = srv.Close()
	}
	slog.Info("server stopped")
}

type tripServer struct {
	api.Unimplemented
	handler *handler.TripHandler
	pool    *pgxpool.Pool
}

func (s *tripServer) Health(w http.ResponseWriter, r *http.Request) {
	handler.Health(w, r)
}

func (s *tripServer) Ready(w http.ResponseWriter, r *http.Request) {
	handler.Ready(w, r, s.pool)
}

func (s *tripServer) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.handler.CreateTrip(w, r, params)
}

func (s *tripServer) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.handler.GetTrip(w, r, tripID)
}

func (s *tripServer) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.handler.FinishTrip(w, r, tripID)
}
