package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/model"
	createUC "github.com/nestera-sh/template/internal/usecase/trip/create"
)

func TestUseCase_Execute(t *testing.T) {
	input := createUC.CreateTripInput{
		Price: 1000,
	}

	tests := []struct {
		name      string
		mockTxErr error
		mockErr   error
		wantErr   error
	}{
		{
			name:      "success",
			mockTxErr: nil,
			mockErr:   nil,
			wantErr:   nil,
		},
		{
			name:      "driver busy",
			mockTxErr: nil,
			mockErr:   trip.ErrDriverBusy,
			wantErr:   trip.ErrDriverBusy,
		},
		{
			name:      "tx rollback error",
			mockTxErr: errors.New("tx error"),
			mockErr:   nil,
			wantErr:   errors.New("tx error"),
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
					if tt.mockTxErr != nil {
						return tt.mockTxErr
					}
					return fn(ctx)
				}).
				Times(1)

			if tt.mockTxErr == nil {
				mockRepo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(tt.mockErr).
					Times(1)

				if tt.mockErr == nil {
					mockRepo.EXPECT().
						SaveHistory(gomock.Any(), gomock.Any(), "", string(model.TripStatusActive)).
						Return(nil).
						Times(1)
				}
			}

			uc := createUC.New(mockRepo, mockTx, nil)
			got, err := uc.Execute(context.Background(), input)

			if tt.wantErr != nil {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, model.TripStatusActive, got.Status)
				assert.Equal(t, input.Price, got.Price)
			}
		})
	}
}
