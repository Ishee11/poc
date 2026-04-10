package app

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	infra "github.com/ishee11/poc/internal/infra"
	postgres "github.com/ishee11/poc/internal/infra/postgres"
	usecase "github.com/ishee11/poc/internal/usecase"
)

func Run() error {
	ctx := context.Background()

	// ===== DB =====
	pool, err := pgxpool.New(ctx, "postgres://user:password@localhost:5432/db")
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

	// ===== Smoke test (вместо HTTP) =====
	err = buyInUC.Execute(usecase.BuyInCommand{
		RequestID: "test-1",
		SessionID: "session-1",
		PlayerID:  "player-1",
		Chips:     100,
	})
	if err != nil {
		log.Println("buy-in error:", err)
	} else {
		log.Println("buy-in success")
	}

	return nil

	// ===== Handler =====
	//handler := httpcontroller.NewHandler(buyInUC) // адаптируешь под себя

	// ===== Router =====
	//router := httpcontroller.NewRouter(handler)

	// ===== HTTP Server =====
	// srv := &http.Server{
	// 	Addr:    ":8080",
	// 	Handler: router,
	// }

	// ===== Start server =====
	// go func() {
	// 	log.Println("server started on :8080")
	// 	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	// 		log.Fatalf("listen error: %v", err)
	// 	}
	// }()

	// ===== Graceful shutdown =====
	// quit := make(chan os.Signal, 1)
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// <-quit
	// log.Println("shutting down server...")

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// if err := srv.Shutdown(ctx); err != nil {
	// 	return err
	// }

	// log.Println("server stopped")
	// return nil
}
