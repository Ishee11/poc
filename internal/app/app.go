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
	"github.com/ishee11/poc/internal/repo/memory"
	sessionuc "github.com/ishee11/poc/internal/usecase/session"
)

func Run() error {
	// ===== Repository =====
	repo := memory.NewSessionRepository()

	// ===== UseCase =====
	uc := sessionuc.NewUseCase(repo)

	// ===== Handler =====
	handler := httpcontroller.NewSessionHandler(uc)

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
