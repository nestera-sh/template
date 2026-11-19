package container

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nestera-sh/template/internal/config"
	"github.com/nestera-sh/template/internal/gateway/push"
	"github.com/nestera-sh/template/internal/infra/postgres/transaction"
	"github.com/nestera-sh/template/internal/infra/telemetry"
	"github.com/nestera-sh/template/internal/repository/trip"
	"github.com/nestera-sh/template/internal/worker/positions"

	createUC "github.com/nestera-sh/template/internal/usecase/trip/create"
	finishUC "github.com/nestera-sh/template/internal/usecase/trip/finish"
	getUC "github.com/nestera-sh/template/internal/usecase/trip/get"
	getPosUC "github.com/nestera-sh/template/internal/usecase/trip/get_positions"
	savePosUC "github.com/nestera-sh/template/internal/usecase/trip/save_position"
)

type Container struct {
	CreateTrip   *createUC.UseCase
	GetTrip      *getUC.UseCase
	FinishTrip   *finishUC.UseCase
	SavePosition *savePosUC.UseCase
	GetPositions *getPosUC.UseCase
	Scanner      *positions.Scanner
}

func New(pool *pgxpool.Pool, cfg *config.Config) *Container {
	txManager := transaction.NewManager(pool)
	tripRepo := trip.NewRepository(pool)
	pushClient := push.NewClient(cfg.PushHTTPURL, cfg.PushRequestTimeout)

	metrics, _ := telemetry.NewMetrics()

	createUc := createUC.New(tripRepo, txManager, metrics)
	getUc := getUC.New(tripRepo)
	finishUc := finishUC.New(tripRepo, txManager, metrics)
	savePosUc := savePosUC.New(tripRepo, txManager)
	getPosUc := getPosUC.New(tripRepo)

	scanner := positions.NewScanner(
		tripRepo,
		pushClient,
		&positions.NoopMetrics{},
		cfg.PositionCheckInterval,
		cfg.PositionStaleAfter,
		cfg.PositionWorkers,
		cfg.PositionQueueSize,
		cfg.PushRequestTimeout,
	)

	return &Container{
		CreateTrip:   createUc,
		GetTrip:      getUc,
		FinishTrip:   finishUc,
		SavePosition: savePosUc,
		GetPositions: getPosUc,
		Scanner:      scanner,
	}
}
