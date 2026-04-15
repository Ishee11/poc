package app

import (
	"context"
	"fmt"
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
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	var (
		pool *pgxpool.Pool
		err  error
	)

	for i := 0; i < 10; i++ {
		pool, err = pgxpool.New(ctx, dsn)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				break
			} else {
				err = pingErr
			}
		}

		log.Println("waiting for db...")
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer pool.Close()

	log.Println("connected to db")

	// ===== MIGRATIONS =====
	if err := postgres.RunMigrations(ctx, pool, postgres.MigrationsFS); err != nil {
		return fmt.Errorf("migrations failed: %w", err)
	}
	log.Println("migrations applied")

	// ===== Repositories =====
	sessionRepo := postgres.NewSessionRepository()
	opRepo := postgres.NewOperationRepository()
	projectionRepo := postgres.NewProjectionRepository()
	idempotencyRepo := postgres.NewIdempotencyRepository()
	statsRepo := postgres.NewStatsRepository()
	playerRepo := postgres.NewPlayerRepository()

	// ===== TxManager =====
	txManager := postgres.NewTxManager(pool)

	// ===== ID Generator =====
	idGen := &infra.UUIDOperationIDGenerator{}

	// ===== UseCases =====

	helper := usecase.NewHelper(
		sessionRepo,
		sessionRepo,
		playerRepo,
		opRepo,
		idGen,
	)

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
		projectionRepo,
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

	getSessionPlayersUC := usecase.NewGetSessionPlayersUseCase(
		playerRepo,
		txManager,
		sessionRepo,
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
		getSessionPlayersUC,
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
