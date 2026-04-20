package app

import (
	"fmt"
	"log/slog"

	"github.com/ishee11/poc/internal/infra"
	"github.com/ishee11/poc/internal/infra/postgres"
	"github.com/ishee11/poc/internal/usecase"
)

func seedAuthAdmin(db *DB, cfg AuthConfig) error {
	if cfg.SeedAdminEmail == "" && cfg.SeedAdminPass == "" {
		return nil
	}

	authRepo := postgres.NewAuthRepository()
	txManager := postgres.NewTxManager(db.Pool)
	seedAdminUC := usecase.NewSeedAdminUseCase(
		authRepo,
		txManager,
		infra.UUIDAuthUserIDGenerator{},
		infra.Argon2IDPasswordHasher{},
		usecase.SystemClock{},
	)

	if err := seedAdminUC.Execute(usecase.SeedAdminCommand{
		Email:    cfg.SeedAdminEmail,
		Password: cfg.SeedAdminPass,
	}); err != nil {
		return fmt.Errorf("seed auth admin: %w", err)
	}

	slog.Info("auth_seed_admin_checked")
	return nil
}
