// @title Poker API
// @version 1.0
// @description Poker sessions service

// @BasePath /
package main

import (
	"log/slog"
	"os"

	"github.com/ishee11/poc/internal/app"
	"github.com/ishee11/poc/pkg/logger"
)

func main() {
	logger.Configure(os.Getenv("LOG_LEVEL"))

	if err := app.Run(); err != nil {
		slog.Error("app_failed", "err", err)
		os.Exit(1)
	}
}
