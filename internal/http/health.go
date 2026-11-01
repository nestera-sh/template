package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const statusKey = "status"

type Pinger interface {
	Ping(ctx context.Context) error
}

func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{statusKey: "ok"})
}

func Ready(w http.ResponseWriter, r *http.Request, db Pinger) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")

	if err := db.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{statusKey: "unavailable"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{statusKey: "ok"})
}
