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

	// ===== Repository =====
	sessionRepo := postgres.NewSessionRepository()
	opRepo := postgres.NewOperationRepository()

	// ===== TxManager =====
	txManager := postgres.NewTxManager(pool)

	// ===== ID Generator =====
	idGen := &infra.UUIDOperationIDGenerator{}

	// ===== UseCases =====
	buyInUC := usecase.NewBuyInUseCase(
		opRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idGen,
	)

	startSessionUC := usecase.NewStartSessionUseCase(
		sessionRepo,
		sessionRepo,
		txManager,
	)

	projectionRepo := postgres.NewProjectionRepository()

	getSessionUC := usecase.NewGetSessionUseCase(
		sessionRepo,
		projectionRepo,
		txManager,
	)

	// ===== Handler =====
	handler := httpcontroller.NewHandler(startSessionUC, buyInUC, getSessionUC)

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("server stopped")
	return nil
}
