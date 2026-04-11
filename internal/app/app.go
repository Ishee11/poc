package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	httpcontroller "github.com/ishee11/poc/internal/controller/http"
	infra "github.com/ishee11/poc/internal/infra"
	postgres "github.com/ishee11/poc/internal/infra/postgres"
	usecase "github.com/ishee11/poc/internal/usecase"
)

func Run() error {
	ctx := context.Background()

	// ===== DB =====
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/poc_test?sslmode=disable")
	if err != nil {
		return err
	}
	defer pool.Close()

	// ===== Repositories =====
	sessionRepo := postgres.NewSessionRepository()
	opRepo := postgres.NewOperationRepository()
	projectionRepo := postgres.NewProjectionRepository()
	idempotencyRepo := postgres.NewIdempotencyRepository()
	statsRepo := postgres.NewStatsRepository()

	// ===== TxManager =====
	txManager := postgres.NewTxManager(pool)

	// ===== ID Generator =====
	idGen := &infra.UUIDOperationIDGenerator{}

	// ===== UseCases =====

	// write
	startSessionUC := usecase.NewStartSessionUseCase(
		sessionRepo,
		sessionRepo,
		txManager,
	)

	buyInUC := usecase.NewBuyInUseCase(
		opRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idGen,
		idempotencyRepo,
	)

	cashOutUC := usecase.NewCashOutUseCase(
		opRepo,
		projectionRepo, // OperationPlayerStateReader
		projectionRepo, // ProjectionRepository
		sessionRepo,
		sessionRepo,
		txManager,
		idGen,
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
		sessionRepo,
		txManager,
		idGen,
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

	getSessionResultsUC := usecase.NewGetSessionResultsUseCase(
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

	// ===== Handler =====
	handler := httpcontroller.NewHandler(
		startSessionUC,
		buyInUC,
		cashOutUC,
		finishSessionUC,
		reverseOpUC,
		getSessionUC,
		getSessionOpsUC,
		getSessionResultsUC,
		getStatsSessionsUC,
		getStatsPlayersUC,
		getPlayerStatsUC,
	)

	// ===== Router =====
	router := httpcontroller.NewRouter(handler)

	// ===== HTTP Server =====
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// ===== Start server =====
	go func() {
		log.Println("server started on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	// ===== Graceful shutdown =====
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("shutting down server...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		return err
	}

	log.Println("server stopped")
	return nil
}
