package get

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=get_test
import (
	"context"

	"github.com/nestera-sh/template/internal/model"
)

type tripRepository interface {
	GetByID(ctx context.Context, id string) (*model.Trip, error)
}
