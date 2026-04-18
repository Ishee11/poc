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
	statsRepo := postgres.NewStatsRepository(db.Pool)
	playerRepo := postgres.NewPlayerRepository()
	debugAdminRepo := postgres.NewDebugAdminRepository()

	// ===== TxManager =====
	txManager := postgres.NewTxManager(db.Pool)

	// ===== Generators =====
	opIDGen := &infra.UUIDOperationIDGenerator{}
	playerIDGen := &infra.UUIDPlayerIDGenerator{}
	sessionIDGen := &infra.UUIDSessionIDGenerator{}

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
		sessionIDGen,
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

	getPlayersUC := usecase.NewGetPlayersUseCase(
		playerRepo,
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

	deleteDebugPlayerUC := usecase.NewDeleteDebugPlayerUseCase(
		debugAdminRepo,
		txManager,
	)

	deleteDebugSessionUC := usecase.NewDeleteDebugSessionUseCase(
		debugAdminRepo,
		txManager,
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
		getPlayersUC,
		getPlayerStatsUC,

		// stats
		getStatsSessionsUC,
		getStatsPlayersUC,

		// debug admin
		deleteDebugPlayerUC,
		deleteDebugSessionUC,
	)

	// ===== Router =====
	router := httpcontroller.NewRouter(handler)

	return &Container{
		Router: router,
	}
}
