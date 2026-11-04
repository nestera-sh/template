package create_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestera-sh/template/internal/domain/trip"
	api "github.com/nestera-sh/template/internal/generated"
	"github.com/nestera-sh/template/internal/http/trip/create"
	"github.com/nestera-sh/template/internal/model"
)

func TestHandler_ServeHTTP(t *testing.T) {
	validBody := api.TripData{
		UserId:     [16]byte{1},
		DriverId:   [16]byte{2},
		StartPoint: api.Coordinates{Latitude: 10, Longitude: 20},
		EndPoint:   api.Coordinates{Latitude: 30, Longitude: 40},
		Price:      1000,
	}

	tests := []struct {
		name            string
		body            interface{}
		mockTrip        *model.Trip
		mockErr         error
		wantStatus      int
		wantProblemCode string
	}{
		{
			name:       "success",
			body:       validBody,
			mockTrip:   &model.Trip{ID: [16]byte{1}, Status: model.TripStatusActive},
			mockErr:    nil,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       "not a json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:            "driver busy",
			body:            validBody,
			mockErr:         trip.ErrDriverBusy,
			wantStatus:      http.StatusConflict,
			wantProblemCode: "driver_busy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := NewMockuseCase(ctrl)

			if tt.wantStatus != http.StatusBadRequest {
				mockUC.EXPECT().
					Execute(gomock.Any(), gomock.Any()).
					Return(tt.mockTrip, tt.mockErr).
					Times(1)
			}

			h := create.New(mockUC)

			var bodyReader *bytes.Reader
			if str, ok := tt.body.(string); ok {
				bodyReader = bytes.NewReader([]byte(str))
			} else {
				b, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewReader(b)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bodyReader)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

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
