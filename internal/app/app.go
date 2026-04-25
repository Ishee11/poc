package app

import (
	"context"

	"github.com/ishee11/poc/pkg/logger"
)

func Run() error {
	ctx := context.Background()

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

	// http server
	server := NewHTTPServer(container.Router, cfg.HTTPPort)

	if err := server.Start(); err != nil {
		return err
	}

	return server.WaitForShutdown()
}
