package finish_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/model"
	finishUC "github.com/nestera-sh/template/internal/usecase/trip/finish"
)

func TestUseCase_Execute(t *testing.T) {
	tests := []struct {
		name       string
		tripStatus model.TripStatus
		getErr     error
		finishErr  error
		wantErr    error
	}{
		{
			name:       "success",
			tripStatus: model.TripStatusActive,
			getErr:     nil,
			finishErr:  nil,
			wantErr:    nil,
		},
		{
			name:       "trip not found",
			tripStatus: model.TripStatusActive,
			getErr:     trip.ErrTripNotFound,
			wantErr:    trip.ErrTripNotFound,
		},
		{
			name:       "trip already completed",
			tripStatus: model.TripStatusCompleted,
			getErr:     nil,
			wantErr:    trip.ErrTripCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMocktripRepository(ctrl)
			mockTx := NewMocktxManager(ctrl)

			mockTx.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).
				Times(1)

			mockRepo.EXPECT().
				GetByID(gomock.Any(), "123").
				Return(&model.Trip{ID: [16]byte{}, Status: tt.tripStatus}, tt.getErr).
				Times(1)

			if tt.getErr == nil && tt.tripStatus == model.TripStatusActive {
				mockRepo.EXPECT().
					Finish(gomock.Any(), gomock.Any()).
					Return(tt.finishErr).
					Times(1)

				mockRepo.EXPECT().
					SaveHistory(gomock.Any(), gomock.Any(), string(model.TripStatusActive), string(model.TripStatusCompleted)).
					Return(nil).
					Times(1)
			}

			uc := finishUC.New(mockRepo, mockTx, nil)
			got, err := uc.Execute(context.Background(), "123")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, model.TripStatusCompleted, got.Status)
			}
		})
	}
}
