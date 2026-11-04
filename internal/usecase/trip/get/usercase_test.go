package get_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestera-sh/template/internal/domain/trip"
	"github.com/nestera-sh/template/internal/model"
	getUC "github.com/nestera-sh/template/internal/usecase/trip/get"
)

func TestUseCase_Execute(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		mockRes *model.Trip
		mockErr error
		wantErr error
	}{
		{
			name:    "success",
			id:      "123",
			mockRes: &model.Trip{},
			mockErr: nil,
			wantErr: nil,
		},
		{
			name:    "trip not found",
			id:      "999",
			mockRes: nil,
			mockErr: trip.ErrTripNotFound,
			wantErr: trip.ErrTripNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMocktripRepository(ctrl)
			mockRepo.EXPECT().
				GetByID(gomock.Any(), tt.id).
				Return(tt.mockRes, tt.mockErr).
				Times(1)

			uc := getUC.New(mockRepo)
			got, err := uc.Execute(context.Background(), tt.id)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.mockRes, got)
			}
		})
	}
}
