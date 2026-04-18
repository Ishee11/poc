package app

import (
	"net/http"

	httpcontroller "github.com/ishee11/poc/internal/controller/http"
	infra "github.com/ishee11/poc/internal/infra"
	postgres "github.com/ishee11/poc/internal/infra/postgres"
	usecase "github.com/ishee11/poc/internal/usecase"
)

type Container struct {
	Router http.Handler
}

// NewContainer — composition root
func NewContainer(db *DB) *Container {
	// ===== Repositories =====
	sessionRepo := postgres.NewSessionRepository()
	opRepo := postgres.NewOperationRepository()
	projectionRepo := postgres.NewProjectionRepository()
	idempotencyRepo := postgres.NewIdempotencyRepository()
	statsRepo := postgres.NewStatsRepository()
	playerRepo := postgres.NewPlayerRepository()

	// ===== TxManager =====
	txManager := postgres.NewTxManager(db.Pool)

	// ===== Generators =====
	opIDGen := &infra.UUIDOperationIDGenerator{}
	playerIDGen := &infra.UUIDPlayerIDGenerator{}

	// ===== Helper =====
	helper := usecase.NewHelper(
		sessionRepo,
		sessionRepo,
		playerRepo,
		opRepo,
		opIDGen,
		playerIDGen,
	)

	// ===== UseCases =====

	// write
	startSessionUC := usecase.NewStartSessionUseCase(
		sessionRepo,
		sessionRepo,
		txManager,
	)

	buyInUC := usecase.NewBuyInUseCase(
		helper,
		txManager,
		idempotencyRepo,
	)

	cashOutUC := usecase.NewCashOutUseCase(
		helper,
		sessionRepo,
		projectionRepo,
		txManager,
		idempotencyRepo,
	)

	finishSessionUC := usecase.NewFinishSessionUseCase(
		projectionRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idempotencyRepo,
	)

	reverseOpUC := usecase.NewReverseOperationUseCase(
		opRepo,
		opRepo,
		opRepo,
		sessionRepo,
		txManager,
		opIDGen,
		idempotencyRepo,
		sessionRepo,
	)

	createPlayerUC := usecase.NewCreatePlayerUseCase(
		helper,
		txManager,
		idempotencyRepo,
	)

	// read
	getSessionUC := usecase.NewGetSessionUseCase(
		sessionRepo,
		projectionRepo,
		txManager,
	)

	getSessionOpsUC := usecase.NewGetSessionOperationsUseCase(
		sessionRepo,
		projectionRepo,
		txManager,
	)

	getStatsSessionsUC := usecase.NewGetStatsSessionsUseCase(
		statsRepo,
		txManager,
	)

	getStatsPlayersUC := usecase.NewGetStatsPlayersUseCase(
		statsRepo,
		txManager,
	)

	getPlayerStatsUC := usecase.NewGetPlayerStatsUseCase(
		statsRepo,
		txManager,
	)

	getSessionPlayersUC := usecase.NewGetSessionPlayersUseCase(
		projectionRepo,
		playerRepo,
		txManager,
		sessionRepo,
	)

	// ===== Handler =====
	handler := httpcontroller.NewHandler(
		// session
		startSessionUC,
		finishSessionUC,
		getSessionUC,
		getSessionPlayersUC,
		getSessionOpsUC,

		// operations
		buyInUC,
		cashOutUC,
		reverseOpUC,

		// player
		createPlayerUC,
		getPlayerStatsUC,

		// stats
		getStatsSessionsUC,
		getStatsPlayersUC,
	)

	// ===== Router =====
	router := httpcontroller.NewRouter(handler)

	return &Container{
		Router: router,
	}
}
