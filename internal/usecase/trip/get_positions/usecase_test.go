package get_positions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nestera-sh/template/internal/model"
	getPosUC "github.com/nestera-sh/template/internal/usecase/trip/get_positions"
)

func TestUseCase_Execute(t *testing.T) {
	mockPositions := []model.Position{
		{ID: 1},
	}

	tests := []struct {
		name    string
		mockRes []model.Position
		mockErr error
		wantErr bool
		wantLen int
	}{
		{
			name:    "success",
			mockRes: mockPositions,
			mockErr: nil,
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "empty",
			mockRes: []model.Position{},
			mockErr: nil,
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "repo error",
			mockRes: nil,
			mockErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMocktripRepository(ctrl)
			mockRepo.EXPECT().
				GetPositions(gomock.Any(), "123").
				Return(tt.mockRes, tt.mockErr).
				Times(1)

			uc := getPosUC.New(mockRepo)
			res, err := uc.Execute(context.Background(), "123")

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.Len(t, res, tt.wantLen)
			}
		})
	}
}
