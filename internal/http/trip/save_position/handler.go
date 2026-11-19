package save_position

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nestera-sh/template/internal/http/domainerror"
	"github.com/nestera-sh/template/internal/usecase/trip/save_position"
)

type useCase interface {
	Execute(ctx context.Context, input save_position.Input) error
}

type Handler struct {
	uc      useCase
	onSaved func(tripID string)
}

func New(uc useCase, onSaved func(tripID string)) *Handler {
	return &Handler{uc: uc, onSaved: onSaved}
}

type savePositionRequest struct {
	Coordinates *struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	} `json:"coordinates"`
	RecordedAt *time.Time `json:"recorded_at"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, tripID string) {
	uid, err := uuid.Parse(tripID)
	if err != nil {
		http.Error(w, "invalid trip id", http.StatusBadRequest)
		return
	}

	var req savePositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.Coordinates == nil || req.Coordinates.Latitude == nil || req.Coordinates.Longitude == nil || req.RecordedAt == nil {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}
	if *req.Coordinates.Latitude < -90 || *req.Coordinates.Latitude > 90 || *req.Coordinates.Longitude < -180 || *req.Coordinates.Longitude > 180 {
		http.Error(w, "coordinates out of range", http.StatusBadRequest)
		return
	}
	if req.RecordedAt.After(time.Now().Add(time.Minute)) {
		http.Error(w, "recorded_at in future", http.StatusBadRequest)
		return
	}

	err = h.uc.Execute(r.Context(), save_position.Input{
		TripID:     uid,
		Latitude:   *req.Coordinates.Latitude,
		Longitude:  *req.Coordinates.Longitude,
		RecordedAt: *req.RecordedAt,
	})
	if err != nil {
		status, prob := domainerror.Problem(err)
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	if h.onSaved != nil {
		h.onSaved(tripID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}
