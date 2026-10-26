package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/nestera-sh/template/internal/config"
	"github.com/nestera-sh/template/internal/container"
	api "github.com/nestera-sh/template/internal/generated"
	httpHealth "github.com/nestera-sh/template/internal/http"
	createHandler "github.com/nestera-sh/template/internal/http/trip/create"
	finishHandler "github.com/nestera-sh/template/internal/http/trip/finish"
	getHandler "github.com/nestera-sh/template/internal/http/trip/get"
	"github.com/nestera-sh/template/internal/infra/postgres"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	pool, err := postgres.New(
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

	cnt := container.New(pool, cfg)

	createH := createHandler.New(cnt.CreateTrip)
	getH := getHandler.New(cnt.GetTrip)
	finishH := finishHandler.New(cnt.FinishTrip)

	mux := chi.NewRouter()
	mux.Get("/health", httpHealth.Health)
	mux.Get("/ready", func(w http.ResponseWriter, r *http.Request) { httpHealth.Ready(w, r, pool) })

	api.HandlerWithOptions(&tripServer{
		createH: createH,
		getH:    getH,
		finishH: finishH,
		pool:    pool,
	}, api.ChiServerOptions{
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(api.Problem{
				Type:     "https://tripgo.example/problems/invalid-request",
				Title:    "Invalid request",
				Status:   http.StatusBadRequest,
				Code:     "invalid_request",
				Detail:   strPtr(err.Error()),
				Instance: strPtr(r.URL.Path),
			})
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
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

// strPtr — вспомогательная функция, чтобы получить указатель на строку.
// Нужна для заполнения полей Detail и Instance в api.Problem.
func strPtr(s string) *string { return &s }

type tripServer struct {
	api.Unimplemented
	createH *createHandler.Handler
	getH    *getHandler.Handler
	finishH *finishHandler.Handler
	pool    interface{ Ping(context.Context) error }
}

func (s *tripServer) Health(w http.ResponseWriter, r *http.Request) { httpHealth.Health(w, r) }
func (s *tripServer) Ready(w http.ResponseWriter, r *http.Request)  { httpHealth.Ready(w, r, s.pool) }

func (s *tripServer) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	s.createH.ServeHTTP(w, r)
}

func (s *tripServer) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.getH.ServeHTTP(w, r, tripID.String())
}

func (s *tripServer) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.finishH.ServeHTTP(w, r, tripID.String())
}
