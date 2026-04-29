package app

import (
	"context"

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
