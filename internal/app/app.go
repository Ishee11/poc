package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpcontroller "github.com/ishee11/poc/internal/controller/http"
	infra "github.com/ishee11/poc/internal/infra"
	postgres "github.com/ishee11/poc/internal/infra/postgres"
	usecase "github.com/ishee11/poc/internal/usecase"
)

func Run() error {
	// ===== Repository =====
	sessionRepo := postgres.NewSessionRepository()
	opRepo := postgres.NewOperationRepository() // <-- нужно добавить

	// ===== TxManager =====
	txManager := postgres.NewTxManager() // <-- если есть, иначе заглушка

	// ===== ID Generator =====
	idGen := &infra.UUIDOperationIDGenerator{}

	// ===== UseCases =====
	buyInUC := usecase.NewBuyInUseCase(
		opRepo,
		sessionRepo,
		txManager,
		idGen,
	)

	// ===== Handler =====
	handler := httpcontroller.NewHandler(buyInUC) // адаптируешь под себя

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
