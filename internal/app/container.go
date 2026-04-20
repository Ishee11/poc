package app

import (
	"net/http"
	"time"

	httpcontroller "github.com/ishee11/poc/internal/controller/http"
	infra "github.com/ishee11/poc/internal/infra"
	postgres "github.com/ishee11/poc/internal/infra/postgres"
	usecase "github.com/ishee11/poc/internal/usecase"
)

type Container struct {
	Router http.Handler
}

// NewContainer — composition root
func NewContainer(db *DB, configs ...*Config) *Container {
	cfg := containerConfig(configs...)

	// ===== Repositories =====
	sessionRepo := postgres.NewSessionRepository()
	opRepo := postgres.NewOperationRepository()
	projectionRepo := postgres.NewProjectionRepository()
	idempotencyRepo := postgres.NewIdempotencyRepository()
	statsRepo := postgres.NewStatsRepository(db.Pool)
	playerRepo := postgres.NewPlayerRepository()
	debugAdminRepo := postgres.NewDebugAdminRepository()
	authRepo := postgres.NewAuthRepository()
	userPlayerLinkRepo := postgres.NewUserPlayerLinkRepository()

	// ===== TxManager =====
	txManager := postgres.NewTxManager(db.Pool)

	// ===== Generators =====
	opIDGen := &infra.UUIDOperationIDGenerator{}
	playerIDGen := &infra.UUIDPlayerIDGenerator{}
	sessionIDGen := &infra.UUIDSessionIDGenerator{}
	authSessionIDGen := infra.UUIDAuthSessionIDGenerator{}
	loginAttemptIDGen := infra.UUIDLoginAttemptIDGenerator{}

	// ===== Helper =====
	helper := usecase.NewHelper(
		sessionRepo,
		sessionRepo,
		playerRepo,
		opRepo,
		opIDGen,
		playerIDGen,
	)

	// ===== UseCases =====

	// write

	startSessionUC := usecase.NewStartSessionUseCase(
		sessionRepo,
		sessionRepo,
		txManager,
		sessionIDGen,
	)

	buyInUC := usecase.NewBuyInUseCase(
		helper,
		txManager,
		idempotencyRepo,
	)

	cashOutUC := usecase.NewCashOutUseCase(
		helper,
		sessionRepo,
		projectionRepo,
		txManager,
		idempotencyRepo,
	)

	finishSessionUC := usecase.NewFinishSessionUseCase(
		projectionRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idempotencyRepo,
	)

	reverseOpUC := usecase.NewReverseOperationUseCase(
		opRepo,
		opRepo,
		opRepo,
		sessionRepo,
		txManager,
		opIDGen,
		idempotencyRepo,
		sessionRepo,
	)

	createPlayerUC := usecase.NewCreatePlayerUseCase(
		helper,
		txManager,
		idempotencyRepo,
	)

	// read
	getSessionUC := usecase.NewGetSessionUseCase(
		sessionRepo,
		projectionRepo,
		txManager,
	)

	getSessionOpsUC := usecase.NewGetSessionOperationsUseCase(
		sessionRepo,
		projectionRepo,
		txManager,
	)

	getStatsSessionsUC := usecase.NewGetStatsSessionsUseCase(
		statsRepo,
		txManager,
	)

	getStatsPlayersUC := usecase.NewGetStatsPlayersUseCase(
		statsRepo,
		txManager,
	)

	getPlayersUC := usecase.NewGetPlayersUseCase(
		playerRepo,
		txManager,
	)

	getPlayerStatsUC := usecase.NewGetPlayerStatsUseCase(
		statsRepo,
		txManager,
	)

	getSessionPlayersUC := usecase.NewGetSessionPlayersUseCase(
		projectionRepo,
		playerRepo,
		txManager,
		sessionRepo,
	)

	renameDebugPlayerUC := usecase.NewRenameDebugPlayerUseCase(
		debugAdminRepo,
		txManager,
	)

	updateDebugSessionConfigUC := usecase.NewUpdateDebugSessionConfigUseCase(
		debugAdminRepo,
		txManager,
	)

	deleteDebugPlayerUC := usecase.NewDeleteDebugPlayerUseCase(
		debugAdminRepo,
		txManager,
	)

	deleteDebugSessionUC := usecase.NewDeleteDebugSessionUseCase(
		debugAdminRepo,
		txManager,
	)

	deleteDebugSessionFinishUC := usecase.NewDeleteDebugSessionFinishUseCase(
		debugAdminRepo,
		txManager,
	)

	passwordHasher := infra.Argon2IDPasswordHasher{}
	authUC := usecase.NewAuthService(
		authRepo,
		authRepo,
		authRepo,
		txManager,
		authSessionIDGen,
		loginAttemptIDGen,
		infra.SecureTokenGenerator{},
		infra.SHA256TokenHasher{},
		passwordHasher,
		usecase.SystemClock{},
		usecase.AuthPolicy{
			SessionTTL:        cfg.Auth.SessionTTL,
			IdleTTL:           cfg.Auth.IdleTTL,
			RateLimitWindow:   usecase.DefaultAuthPolicy().RateLimitWindow,
			MaxFailedAttempts: usecase.DefaultAuthPolicy().MaxFailedAttempts,
		},
	)
	userPlayerLinksUC := usecase.NewUserPlayerLinksUseCase(
		userPlayerLinkRepo,
		playerRepo,
		txManager,
	)

	// ===== Handler =====
	handler := httpcontroller.NewHandler(
		httpcontroller.AuthCookieConfig{
			Name:     cfg.Auth.CookieName,
			Secure:   cfg.Auth.CookieSecure,
			SameSite: sameSite(cfg.Auth.CookieSameSite),
			MaxAge:   cfg.Auth.SessionTTL,
		},
		authUC,
		userPlayerLinksUC,

		// session
		startSessionUC,
		finishSessionUC,
		getSessionUC,
		getSessionPlayersUC,
		getSessionOpsUC,

		// operations
		buyInUC,
		cashOutUC,
		reverseOpUC,

		// player
		createPlayerUC,
		getPlayersUC,
		getPlayerStatsUC,

		// stats
		getStatsSessionsUC,
		getStatsPlayersUC,

		// debug admin
		renameDebugPlayerUC,
		updateDebugSessionConfigUC,
		deleteDebugPlayerUC,
		deleteDebugSessionUC,
		deleteDebugSessionFinishUC,
	)

	// ===== Router =====
	router := httpcontroller.NewRouter(handler)

	return &Container{
		Router: router,
	}
}

func containerConfig(configs ...*Config) *Config {
	if len(configs) > 0 && configs[0] != nil {
		return configs[0]
	}

	return &Config{
		Auth: AuthConfig{
			CookieName:     "sid",
			CookieSecure:   true,
			CookieSameSite: "Lax",
			SessionTTL:     12 * time.Hour,
			IdleTTL:        2 * time.Hour,
		},
	}
}

func sameSite(value string) http.SameSite {
	switch value {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
