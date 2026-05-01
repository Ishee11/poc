package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"

	"github.com/ishee11/poc/internal/app"
	kafkainfra "github.com/ishee11/poc/internal/infra/kafka"
	postgresinfra "github.com/ishee11/poc/internal/infra/postgres"
	"github.com/ishee11/poc/pkg/logger"
)

func main() {
	logger.Configure(os.Getenv("LOG_LEVEL"))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, kafka.ErrGenerationEnded) {
		slog.Error("audit_consumer_failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := app.Load()
	if err != nil {
		return err
	}
	logger.Configure(cfg.LogLevel)

	db, err := app.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	repo := postgresinfra.NewAuditRepository(db.Pool)
	consumer := kafkainfra.NewAuditConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.OutboxTopic,
		cfg.Kafka.AuditGroupID,
		repo,
	)
	defer func() { _ = consumer.Close() }()

	slog.Info("audit_consumer_started", "topic", cfg.Kafka.OutboxTopic, "group_id", cfg.Kafka.AuditGroupID)
	return consumer.Run(ctx)
}
