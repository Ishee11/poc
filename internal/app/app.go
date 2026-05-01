package app

import (
	"context"
	"time"

	"github.com/ishee11/poc/pkg/logger"
)

func Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// config
	cfg, err := Load()
	if err != nil {
		return err
	}
	logger.Configure(cfg.LogLevel)
	shutdownTracing, err := setupTracing(ctx, cfg.Tracing)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = shutdownTracing(shutdownCtx)
	}()

	// db
	db, err := NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := seedAuthUsers(db, cfg.Auth); err != nil {
		return err
	}

	// container (DI)
	container := NewContainer(db, cfg)
	if container.PushNotifier != nil {
		container.PushNotifier.Start(ctx)
	}
	if container.OutboxRelay != nil {
		container.OutboxRelay.Start(ctx)
	}

	// http server
	server := NewHTTPServer(container.Router, cfg.HTTPPort)

	if err := server.Start(); err != nil {
		return err
	}

	return server.WaitForShutdown()
}
