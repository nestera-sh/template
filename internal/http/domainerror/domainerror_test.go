package domainerror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "trip not found",
			err:        trip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "trip completed",
			err:        trip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "driver busy",
			err:        trip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name:       "unknown error",
			err:        errors.New("some internal db error"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, prob := domainerror.Problem(tt.err)

			if status != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, status)
			}
			if prob.Code != tt.wantCode {
				t.Errorf("expected code %q, got %q", tt.wantCode, prob.Code)
			}

			if tt.wantCode == "internal_error" && prob.Detail != nil && *prob.Detail != "Internal server error" {
				t.Errorf("internal error details leaked: %v", *prob.Detail)
			}
		})
	}
}
