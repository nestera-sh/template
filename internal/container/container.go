package container

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nestera-sh/template/internal/config"
	"github.com/nestera-sh/template/internal/infra/postgres/transaction"
	"github.com/nestera-sh/template/internal/infra/telemetry"
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

func New(pool *pgxpool.Pool, _ *config.Config) *Container {
	txManager := transaction.NewManager(pool)
	tripRepo := trip.NewRepository(pool)

	metrics, _ := telemetry.NewMetrics()

	createUc := createUC.New(tripRepo, txManager, metrics)
	getUc := getUC.New(tripRepo)
	finishUc := finishUC.New(tripRepo, txManager, metrics)

	return &Container{
		CreateTrip: createUc,
		GetTrip:    getUc,
		FinishTrip: finishUc,
	}
}
