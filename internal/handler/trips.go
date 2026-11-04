package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	api "github.com/nestera-sh/template/internal/generated"
	"github.com/nestera-sh/template/internal/model"
	"github.com/nestera-sh/template/internal/repository"
	"github.com/nestera-sh/template/internal/txmanager"
)

type TripHandler struct {
	tripRepo     *repository.TripRepository
	historyRepo  *repository.HistoryRepository
	txManager    *txmanager.TxManager
	queryTimeout time.Duration
}

func NewTripHandler(
	tripRepo *repository.TripRepository,
	historyRepo *repository.HistoryRepository,
	txManager *txmanager.TxManager,
	queryTimeout time.Duration,
) *TripHandler {
	return &TripHandler{
		tripRepo:     tripRepo,
		historyRepo:  historyRepo,
		txManager:    txManager,
		queryTimeout: queryTimeout,
	}
}

func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {

	var req api.TripData

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondBadRequest(w, r, "Malformed JSON body")
		return
	}

	if req.UserId == (uuid.UUID{}) || req.DriverId == (uuid.UUID{}) {
		respondBadRequest(w, r, "user_id and driver_id must be non-empty UUIDs")
		return
	}

	if !validCoords(req.StartPoint) || !validCoords(req.EndPoint) {
		respondBadRequest(w, r, "coordinates are out of range")
		return
	}

	if req.Price < 0 {
		respondBadRequest(w, r, "price must be >= 0")
		return
	}

	trip := &model.Trip{
		ID:             uuid.New(),
		UserID:         req.UserId,
		DriverID:       req.DriverId,
		StartLatitude:  req.StartPoint.Latitude,
		StartLongitude: req.StartPoint.Longitude,
		EndLatitude:    req.EndPoint.Latitude,
		EndLongitude:   req.EndPoint.Longitude,
		Price:          req.Price,
		Status:         model.TripStatusActive,
		StartedAt:      time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)

	defer cancel()

	err := h.txManager.Do(ctx, func(ctx context.Context) error {
		if err := h.tripRepo.Create(ctx, trip); err != nil {
			return err
		}
		return h.historyRepo.Insert(ctx, trip.ID, "", string(model.TripStatusActive))
	})

	if err != nil {
		h.handleTripError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	respondJSON(w, http.StatusCreated, toAPITrip(trip))
}

func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	trip, err := h.tripRepo.GetByID(ctx, tripID)
	if err != nil {
		h.handleTripError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, toAPITrip(trip))
}

func (h *TripHandler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	err := h.txManager.Do(ctx, func(ctx context.Context) error {
		if _, err := h.tripRepo.Finish(ctx, tripID); err != nil {
			return err
		}
		return h.historyRepo.Insert(ctx, tripID, string(model.TripStatusActive), string(model.TripStatusCompleted))
	})

	if err != nil {
		h.handleTripError(w, r, err)
		return
	}

	trip, err := h.tripRepo.GetByID(ctx, tripID)
	if err != nil {
		h.handleTripError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, toAPITrip(trip))
}

func (h *TripHandler) handleTripError(w http.ResponseWriter, r *http.Request, err error) {

	switch {
	case errors.Is(err, model.ErrTripNotFound):
		respondTripNotFound(w, r)
	case errors.Is(err, model.ErrTripCompleted):
		respondTripCompleted(w, r)
	case errors.Is(err, model.ErrDriverBusy):
		respondDriverBusy(w, r)
	default:
		slog.Error("trip handler", "err", err, "path", r.URL.Path)
		respondInternalError(w, r)
	}
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

func validCoords(c api.Coordinates) bool {

	return c.Latitude >= -90 && c.Latitude <= 90 &&
		c.Longitude >= -180 && c.Longitude <= 180
}
