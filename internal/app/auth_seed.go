package app

import (
	"fmt"
	"log/slog"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/infra"
	"github.com/ishee11/poc/internal/infra/postgres"
	"github.com/ishee11/poc/internal/usecase"
)

func seedAuthUsers(db *DB, cfg AuthConfig) error {
	if cfg.SeedAdminEmail == "" && cfg.SeedAdminPass == "" &&
		cfg.SeedUserEmail == "" && cfg.SeedUserPass == "" {
		return nil
	}

	authRepo := postgres.NewAuthRepository()
	txManager := postgres.NewTxManager(db.Pool)
	seedUserUC := usecase.NewSeedUserUseCase(
		authRepo,
		txManager,
		infra.UUIDAuthUserIDGenerator{},
		infra.Argon2IDPasswordHasher{},
		usecase.SystemClock{},
	)

	if err := seedUserUC.Execute(usecase.SeedUserCommand{
		Email:    cfg.SeedAdminEmail,
		Password: cfg.SeedAdminPass,
		Role:     entity.AuthRoleAdmin,
	}); err != nil {
		return fmt.Errorf("seed auth admin: %w", err)
	}

	if err := seedUserUC.Execute(usecase.SeedUserCommand{
		Email:    cfg.SeedUserEmail,
		Password: cfg.SeedUserPass,
		Role:     entity.AuthRoleUser,
	}); err != nil {
		return fmt.Errorf("seed auth user: %w", err)
	}

	slog.Info("auth_seed_users_checked")
	return nil
}
