package get_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestera-sh/template/internal/domain/trip"
	api "github.com/nestera-sh/template/internal/generated"
	"github.com/nestera-sh/template/internal/http/trip/get"
	"github.com/nestera-sh/template/internal/model"
)

func TestHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name            string
		mockTrip        *model.Trip
		mockErr         error
		wantStatus      int
		wantProblemCode string
	}{
		{
			name:       "success",
			mockTrip:   &model.Trip{ID: [16]byte{1}, Status: model.TripStatusActive},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:            "trip not found",
			mockTrip:        nil,
			mockErr:         trip.ErrTripNotFound,
			wantStatus:      http.StatusNotFound,
			wantProblemCode: "trip_not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := NewMockuseCase(ctrl)
			mockUC.EXPECT().
				Execute(gomock.Any(), "123").
				Return(tt.mockTrip, tt.mockErr).
				Times(1)

			h := get.New(mockUC)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/123", nil)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req, "123")

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantProblemCode != "" {
				assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
				var prob api.Problem
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&prob))
				assert.Equal(t, tt.wantProblemCode, prob.Code)
			}
		})
	}
}
