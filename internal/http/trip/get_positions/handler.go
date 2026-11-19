package get_positions

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/nestera-sh/template/internal/http/domainerror"
	"github.com/nestera-sh/template/internal/model"
)

type useCase interface {
	Execute(ctx context.Context, tripID string) ([]model.Position, error)
}

type Handler struct {
	uc useCase
}

func New(uc useCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, tripID string) {
	positions, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		status, prob := domainerror.Problem(err)
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	resp := struct {
		Positions []model.Position `json:"positions"`
	}{Positions: positions}
	if resp.Positions == nil {
		resp.Positions = []model.Position{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
