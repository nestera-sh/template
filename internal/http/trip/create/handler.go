package create

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	api "github.com/nestera-sh/template/internal/generated"
	"github.com/nestera-sh/template/internal/http/domainerror"
	"github.com/nestera-sh/template/internal/model"
	"github.com/nestera-sh/template/internal/usecase/trip/create"
)

//go:generate mockgen -source=handler.go -destination=mocks_test.go -package=create_test

type useCase interface {
	Execute(ctx context.Context, input create.CreateTripInput) (*model.Trip, error)
}

type Handler struct {
	uc useCase
}

func New(uc useCase) *Handler {
	return &Handler{uc: uc}
}

// для строгой валидации  JSON
type createTripRequest struct {
	UserID     *uuid.UUID `json:"user_id"`
	DriverID   *uuid.UUID `json:"driver_id"`
	StartPoint *struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	} `json:"start_point"`
	EndPoint *struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	} `json:"end_point"`
	Price *int64 `json:"price"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req createTripRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Malformed JSON body or unknown fields", http.StatusBadRequest)
		return
	}

	if req.UserID == nil || req.DriverID == nil || req.Price == nil || req.StartPoint == nil || req.EndPoint == nil {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	if req.StartPoint.Latitude == nil || req.StartPoint.Longitude == nil || req.EndPoint.Latitude == nil || req.EndPoint.Longitude == nil {
		http.Error(w, "Missing coordinates", http.StatusBadRequest)
		return
	}
	if *req.UserID == uuid.Nil || *req.DriverID == uuid.Nil {
		http.Error(w, "user_id and driver_id must be non-empty UUIDs", http.StatusBadRequest)
		return
	}
	if !validCoords(*req.StartPoint.Latitude, *req.StartPoint.Longitude) || !validCoords(*req.EndPoint.Latitude, *req.EndPoint.Longitude) {
		http.Error(w, "coordinates are out of range", http.StatusBadRequest)
		return
	}
	if *req.Price < 0 {
		http.Error(w, "price must be >= 0", http.StatusBadRequest)
		return
	}

	t, err := h.uc.Execute(r.Context(), create.CreateTripInput{
		UserID:         *req.UserID,
		DriverID:       *req.DriverID,
		StartLatitude:  *req.StartPoint.Latitude,
		StartLongitude: *req.StartPoint.Longitude,
		EndLatitude:    *req.EndPoint.Latitude,
		EndLongitude:   *req.EndPoint.Longitude,
		Price:          *req.Price,
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

func validCoords(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

//тардис
