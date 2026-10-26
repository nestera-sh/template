package container

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nestera-sh/template/internal/config"
	"github.com/nestera-sh/template/internal/infra/postgres/transaction"
	"github.com/nestera-sh/template/internal/repository/trip"
	createUC "github.com/nestera-sh/template/internal/usecase/trip/create"
	finishUC "github.com/nestera-sh/template/internal/usecase/trip/finish"
	getUC "github.com/nestera-sh/template/internal/usecase/trip/get"
)

type Container struct {
	CreateTrip *createUC.UseCase
	GetTrip    *getUC.UseCase
	FinishTrip *finishUC.UseCase
}

func New(pool *pgxpool.Pool, cfg *config.Config) *Container {

	txManager := transaction.NewManager(pool)

	tripRepo := trip.NewRepository(pool)

	createUc := createUC.New(tripRepo, txManager)
	getUc := getUC.New(tripRepo)
	finishUc := finishUC.New(tripRepo, txManager)

	return &Container{
		CreateTrip: createUc,
		GetTrip:    getUc,
		FinishTrip: finishUc,
	}
}
