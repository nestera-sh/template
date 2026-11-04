package get

import (
	"context"
	"encoding/json"
	"net/http"

	api "github.com/nestera-sh/template/internal/generated"
	"github.com/nestera-sh/template/internal/http/domainerror"
	"github.com/nestera-sh/template/internal/model"
)

//go:generate mockgen -source=handler.go -destination=mocks_test.go -package=get_test

type useCase interface {
	Execute(ctx context.Context, id string) (*model.Trip, error)
}

type Handler struct {
	uc useCase
}

func New(uc useCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.uc.Execute(r.Context(), id)
	if err != nil {
		status, prob := domainerror.Problem(err)
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
