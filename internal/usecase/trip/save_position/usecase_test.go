package save_position_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	savePosUC "github.com/nestera-sh/template/internal/usecase/trip/save_position"
)

func TestUseCase_Execute(t *testing.T) {
	input := savePosUC.Input{
		TripID:     uuid.New(),
		Latitude:   59.9,
		Longitude:  30.3,
		RecordedAt: time.Now(),
	}

	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{
			name:    "success",
			mockErr: nil,
			wantErr: false,
		},
		{
			name:    "repo error",
			mockErr: errors.New("db error"),
			wantErr: true,
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
				SavePosition(gomock.Any(), gomock.Any()).
				Return(tt.mockErr).
				Times(1)

			uc := savePosUC.New(mockRepo, mockTx)
			err := uc.Execute(context.Background(), input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
