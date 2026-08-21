package http

import (
	"net/http"
	"time"

	"github.com/ishee11/poc/internal/usecase"
)

type Handler struct {
	Auth        *AuthHandler
	Account     *AccountHandler
	Session     *SessionHandler
	Operation   *OperationHandler
	Blinds      *BlindClockHandler
	Push        *PushHandler
	Player      *PlayerHandler
	Stats       *StatsHandler
	Admin       *AdminHandler
	TelegramBot *TelegramBotHandler
}

type AuthCookieConfig struct {
	Enabled  bool
	Name     string
	Secure   bool
	SameSite http.SameSite
	MaxAge   time.Duration
}

func NewHandler(
	authCookie AuthCookieConfig,
	authUC *usecase.AuthService,
	registerUserUC *usecase.RegisterUserUseCase,
	telegramAuthUC *usecase.TelegramAuthService,
	telegramChallengeUC *usecase.TelegramLoginChallengeService,
	telegramBot TelegramBotSender,
	telegramBotWebhookSecret string,
	userPlayerLinksUC *usecase.UserPlayerLinksUseCase,
	sessionAccessUC *usecase.SessionAccessService,

	// session
	startSession *usecase.StartSessionUseCase,
	finishSession *usecase.FinishSessionUseCase,
	getSession *usecase.GetSessionUseCase,
	getSessionPlayers *usecase.GetSessionPlayersUseCase,
	getSessionOps *usecase.GetSessionOperationsUseCase,

	// operations
	buyIn *usecase.BuyInUseCase,
	cashOut *usecase.CashOutUseCase,
	reverse *usecase.ReverseOperationUseCase,
	sessionExpenses *usecase.SessionExpenseService,
	settlementTransfers *usecase.SettlementTransferService,

	// blinds
	blindClockUC *usecase.BlindClockService,
	pushUC *usecase.BlindClockPushService,

	// player
	createPlayer *usecase.CreatePlayerUseCase,
	getPlayers *usecase.GetPlayersUseCase,
	getPlayerStats *usecase.GetPlayerStatsUseCase,

	// stats
	getStatsSessions *usecase.GetStatsSessionsUseCase,
	getStatsPlayers *usecase.GetStatsPlayersUseCase,

	// admin
	renameAdminPlayer *usecase.AdminRenamePlayerUseCase,
	updateAdminSessionConfig *usecase.AdminUpdateSessionConfigUseCase,
	deleteAdminPlayer *usecase.AdminDeletePlayerUseCase,
	deleteAdminSession *usecase.AdminDeleteSessionUseCase,
	deleteAdminSessionFinish *usecase.AdminDeleteSessionFinishUseCase,
	adminAccountOwnership *usecase.AdminAccountOwnershipService,
) *Handler {
	access := &sessionAccessAuthorizer{
		service: sessionAccessUC,
		authUC:  authUC,
		cookie:  authCookie,
	}

	return &Handler{
		Auth: &AuthHandler{
			authUC:              authUC,
			registerUserUC:      registerUserUC,
			telegramAuthUC:      telegramAuthUC,
			telegramChallengeUC: telegramChallengeUC,
			cookie:              authCookie,
		},
		TelegramBot: &TelegramBotHandler{service: telegramChallengeUC, sender: telegramBot, webhookSecret: telegramBotWebhookSecret},
		Account: &AccountHandler{
			authUC:            authUC,
			telegramAuthUC:    telegramAuthUC,
			userPlayerLinksUC: userPlayerLinksUC,
			cookie:            authCookie,
		},
		Session: &SessionHandler{
			startSessionUC:      startSession,
			finishSessionUC:     finishSession,
			getSessionUC:        getSession,
			getSessionPlayersUC: getSessionPlayers,
			getSessionOpsUC:     getSessionOps,
			access:              access,
		},
		Operation: &OperationHandler{
			buyInUC:                   buyIn,
			cashOutUC:                 cashOut,
			reverseOperationUC:        reverse,
			expenseService:            sessionExpenses,
			settlementTransferService: settlementTransfers,
			authUC:                    authUC,
			cookie:                    authCookie,
			access:                    access,
		},
		Blinds: &BlindClockHandler{
			service: blindClockUC,
		},
		Push: &PushHandler{
			service: pushUC,
		},
		Player: &PlayerHandler{
			createPlayerUC:   createPlayer,
			getPlayersUC:     getPlayers,
			getPlayerStatsUC: getPlayerStats,
			access:           access,
		},
		Stats: &StatsHandler{
			getStatsSessionsUC: getStatsSessions,
			getStatsPlayersUC:  getStatsPlayers,
			authUC:             authUC,
			cookie:             authCookie,
			access:             access,
		},
		Admin: &AdminHandler{
			renamePlayerUC:        renameAdminPlayer,
			updateSessionConfigUC: updateAdminSessionConfig,
			deletePlayerUC:        deleteAdminPlayer,
			deleteSessionUC:       deleteAdminSession,
			deleteSessionFinishUC: deleteAdminSessionFinish,
			accountOwnershipUC:    adminAccountOwnership,
			authUC:                authUC,
			cookie:                authCookie,
		},
	}
}

type AuthHandler struct {
	authUC              *usecase.AuthService
	registerUserUC      *usecase.RegisterUserUseCase
	telegramAuthUC      *usecase.TelegramAuthService
	telegramChallengeUC *usecase.TelegramLoginChallengeService
	cookie              AuthCookieConfig
}

type AccountHandler struct {
	authUC            *usecase.AuthService
	telegramAuthUC    *usecase.TelegramAuthService
	userPlayerLinksUC *usecase.UserPlayerLinksUseCase
	cookie            AuthCookieConfig
}

type SessionHandler struct {
	startSessionUC      *usecase.StartSessionUseCase
	finishSessionUC     *usecase.FinishSessionUseCase
	getSessionUC        *usecase.GetSessionUseCase
	getSessionPlayersUC *usecase.GetSessionPlayersUseCase
	getSessionOpsUC     *usecase.GetSessionOperationsUseCase
	access              *sessionAccessAuthorizer
}

type OperationHandler struct {
	buyInUC                   *usecase.BuyInUseCase
	cashOutUC                 *usecase.CashOutUseCase
	reverseOperationUC        *usecase.ReverseOperationUseCase
	expenseService            *usecase.SessionExpenseService
	settlementTransferService *usecase.SettlementTransferService
	authUC                    *usecase.AuthService
	cookie                    AuthCookieConfig
	access                    *sessionAccessAuthorizer
}

type BlindClockHandler struct {
	service *usecase.BlindClockService
}

type PushHandler struct {
	service *usecase.BlindClockPushService
}

type PlayerHandler struct {
	createPlayerUC   *usecase.CreatePlayerUseCase
	getPlayersUC     *usecase.GetPlayersUseCase
	getPlayerStatsUC *usecase.GetPlayerStatsUseCase
	access           *sessionAccessAuthorizer
}

type StatsHandler struct {
	getStatsSessionsUC *usecase.GetStatsSessionsUseCase
	getStatsPlayersUC  *usecase.GetStatsPlayersUseCase
	authUC             *usecase.AuthService
	cookie             AuthCookieConfig
	access             *sessionAccessAuthorizer
}

type AdminHandler struct {
	renamePlayerUC        *usecase.AdminRenamePlayerUseCase
	updateSessionConfigUC *usecase.AdminUpdateSessionConfigUseCase
	deletePlayerUC        *usecase.AdminDeletePlayerUseCase
	deleteSessionUC       *usecase.AdminDeleteSessionUseCase
	deleteSessionFinishUC *usecase.AdminDeleteSessionFinishUseCase
	accountOwnershipUC    *usecase.AdminAccountOwnershipService
	authUC                *usecase.AuthService
	cookie                AuthCookieConfig
}
