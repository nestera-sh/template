package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/model"
)

func TestTrip_Finish(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name       string
		status     model.TripStatus
		wantErr    error
		wantStatus model.TripStatus
	}{
		{
			name:       "active trip finishes successfully",
			status:     model.TripStatusActive,
			wantErr:    nil,
			wantStatus: model.TripStatusCompleted,
		},
		{
			name:       "already completed trip returns error",
			status:     model.TripStatusCompleted,
			wantErr:    trip.ErrTripCompleted,
			wantStatus: model.TripStatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tripModel := &model.Trip{
				Status: tt.status,
			}

			err := trip.Finish(tripModel, now)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if tripModel.Status != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, tripModel.Status)
			}

			if tt.status == model.TripStatusActive {
				if tripModel.FinishedAt == nil {
					t.Error("expected FinishedAt to be set, got nil")
				} else if !tripModel.FinishedAt.Equal(now) {
					t.Errorf("expected FinishedAt %v, got %v", now, *tripModel.FinishedAt)
				}
			} else {
				if tripModel.FinishedAt != nil {
					t.Error("expected FinishedAt to be nil for already completed trip")
				}
			}
		})
	}
}
