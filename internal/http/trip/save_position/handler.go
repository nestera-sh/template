package save_position

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/nestera-sh/template/internal/http/domainerror"
	"github.com/nestera-sh/template/internal/model"
	savePositionUC "github.com/nestera-sh/template/internal/usecase/trip/save_position"
)

type useCase interface {
	Execute(ctx context.Context, input savePositionUC.Input) (*model.Position, error)
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
		domainerror.WriteInvalidRequest(w, r, "invalid trip id")
		return
	}

	var req savePositionRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		domainerror.WriteInvalidRequest(w, r, "malformed JSON body")
		return
	}

	if req.Coordinates == nil || req.Coordinates.Latitude == nil || req.Coordinates.Longitude == nil || req.RecordedAt == nil {
		domainerror.WriteInvalidRequest(w, r, "missing required fields")
		return
	}
	if *req.Coordinates.Latitude < -90 || *req.Coordinates.Latitude > 90 ||
		*req.Coordinates.Longitude < -180 || *req.Coordinates.Longitude > 180 {
		domainerror.WriteInvalidRequest(w, r, "coordinates out of range")
		return
	}
	if req.RecordedAt.After(time.Now().Add(time.Minute)) {
		domainerror.WriteInvalidRequest(w, r, "recorded_at in future")
		return
	}

	pos, err := h.uc.Execute(r.Context(), savePositionUC.Input{
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

	resp := struct {
		ID          int64             `json:"id"`
		TripID      uuid.UUID         `json:"trip_id"`
		Coordinates model.Coordinates `json:"coordinates"`
		RecordedAt  time.Time         `json:"recorded_at"`
	}{
		ID:          pos.ID,
		TripID:      pos.TripID,
		Coordinates: pos.Coordinates,
		RecordedAt:  pos.RecordedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

var _ = errors.Is
