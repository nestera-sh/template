package save_position_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/nestera-sh/template/internal/domain/trip"
	savePositionHTTP "github.com/nestera-sh/template/internal/http/trip/save_position"
	"github.com/nestera-sh/template/internal/model"
	savePosUC "github.com/nestera-sh/template/internal/usecase/trip/save_position"
)

type stubUseCase struct {
	result *model.Position
	err    error
	calls  int
}

func (s *stubUseCase) Execute(
	_ context.Context,
	_ savePosUC.Input,
) (*model.Position, error) {
	s.calls++
	return s.result, s.err
}

func TestHandler_ServeHTTP(t *testing.T) {
	tripID := uuid.New()
	recorded := time.Now().UTC().Add(-time.Second)

	validBody := map[string]any{
		"coordinates": map[string]float64{
			"latitude":  59.9,
			"longitude": 30.3,
		},
		"recorded_at": recorded.Format(time.RFC3339Nano),
	}

	tests := []struct {
		name       string
		tripID     string
		body       any
		mockRes    *model.Position
		mockErr    error
		wantStatus int
		wantCalls  int
	}{
		{
			name:   "success",
			tripID: tripID.String(),
			body:   validBody,
			mockRes: &model.Position{
				ID:     42,
				TripID: tripID,
				Coordinates: model.Coordinates{
					Latitude:  59.9,
					Longitude: 30.3,
				},
				RecordedAt: recorded,
			},
			wantStatus: http.StatusCreated,
			wantCalls:  1,
		},
		{
			name:       "bad trip id",
			tripID:     "not-a-uuid",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "missing coordinates",
			tripID: tripID.String(),
			body: map[string]any{
				"recorded_at": recorded.Format(time.RFC3339Nano),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "trip completed",
			tripID:     tripID.String(),
			body:       validBody,
			mockErr:    trip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCalls:  1,
		},
		{
			name:       "trip not found",
			tripID:     tripID.String(),
			body:       validBody,
			mockErr:    trip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCalls:  1,
		},
		{
			name:       "repository error",
			tripID:     tripID.String(),
			body:       validBody,
			mockErr:    errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &stubUseCase{
				result: tt.mockRes,
				err:    tt.mockErr,
			}

			h := savePositionHTTP.New(uc, nil)

			body, err := json.Marshal(tt.body)
			assert.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/trips/"+tt.tripID+"/positions",
				bytes.NewReader(body),
			)

			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req, tt.tripID)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantCalls, uc.calls)

			if tt.wantStatus == http.StatusCreated {
				assert.Equal(
					t,
					"application/json",
					rec.Header().Get("Content-Type"),
				)
			}

			if tt.wantStatus >= http.StatusBadRequest {
				assert.Equal(
					t,
					"application/problem+json",
					rec.Header().Get("Content-Type"),
				)
			}
		})
	}
}
