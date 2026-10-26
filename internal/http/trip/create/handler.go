package create

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	api "github.com/nestera-sh/template/internal/generated"
	"github.com/nestera-sh/template/internal/http/domainerror"
	"github.com/nestera-sh/template/internal/model"
	"github.com/nestera-sh/template/internal/usecase/trip/create"
)

type Handler struct {
	uc *create.UseCase
}

func New(uc *create.UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req api.TripData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.UserId == uuid.Nil || req.DriverId == uuid.Nil || req.Price < 0 {
		http.Error(w, "validation failed", http.StatusBadRequest)
		return
	}

	t, err := h.uc.Execute(r.Context(), create.CreateTripInput{
		UserID:         req.UserId,
		DriverID:       req.DriverId,
		StartLatitude:  req.StartPoint.Latitude,
		StartLongitude: req.StartPoint.Longitude,
		EndLatitude:    req.EndPoint.Latitude,
		EndLongitude:   req.EndPoint.Longitude,
		Price:          req.Price,
	})

	if err != nil {
		status, prob := domainerror.Problem(err)
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+t.ID.String())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAPITrip(t))
}

func toAPITrip(t *model.Trip) api.Trip {
	return api.Trip{
		Id:             t.ID,
		UserId:         t.UserID,
		DriverId:       t.DriverID,
		StartPoint:     api.Coordinates{Latitude: t.StartLatitude, Longitude: t.StartLongitude},
		EndPoint:       api.Coordinates{Latitude: t.EndLatitude, Longitude: t.EndLongitude},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: t.LastPositionAt,
	}
}
