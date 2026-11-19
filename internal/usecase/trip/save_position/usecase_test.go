package save_position_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestera-sh/template/internal/model"
	savePosUC "github.com/nestera-sh/template/internal/usecase/trip/save_position"
)

func TestUseCase_Execute(t *testing.T) {
	input := savePosUC.Input{
		TripID:     uuid.New(),
		Latitude:   59.9,
		Longitude:  30.3,
		RecordedAt: time.Now().UTC(),
	}

	tests := []struct {
		name    string
		trip    *model.Trip
		getErr  error
		saveErr error
		wantErr bool
	}{
		{
			name: "success",
			trip: &model.Trip{
				Status: model.TripStatusActive,
			},
		},
		{
			name:    "repository get error",
			getErr:  errors.New("database error"),
			wantErr: true,
		},
		{
			name: "repository save error",
			trip: &model.Trip{
				Status: model.TripStatusActive,
			},
			saveErr: errors.New("save position error"),
			wantErr: true,
		},
		{
			name: "completed trip",
			trip: &model.Trip{
				Status: model.TripStatusCompleted,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockRepo := NewMocktripRepository(ctrl)
			mockTx := NewMocktxManager(ctrl)

			mockTx.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				DoAndReturn(func(
					ctx context.Context,
					fn func(context.Context) error,
				) error {
					return fn(ctx)
				}).
				Times(1)

			mockRepo.EXPECT().
				GetByID(gomock.Any(), input.TripID.String()).
				Return(tt.trip, tt.getErr).
				Times(1)

			if tt.trip != nil && tt.trip.Status == model.TripStatusActive {
				mockRepo.EXPECT().
					SavePosition(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						position *model.Position,
					) error {
						require.Equal(t, input.TripID, position.TripID)
						require.Equal(t, input.Latitude, position.Coordinates.Latitude)
						require.Equal(t, input.Longitude, position.Coordinates.Longitude)
						require.Equal(t, input.RecordedAt, position.RecordedAt)

						return tt.saveErr
					}).
					Times(1)
			}

			uc := savePosUC.New(mockRepo, mockTx, nil)

			_, err := uc.Execute(context.Background(), input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
