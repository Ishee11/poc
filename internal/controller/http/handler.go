package http

import (
	"net/http"
	"time"

	"github.com/ishee11/poc/internal/usecase"
)

type Handler struct {
	Auth      *AuthHandler
	Account   *AccountHandler
	Session   *SessionHandler
	Operation *OperationHandler
	Blinds    *BlindClockHandler
	Player    *PlayerHandler
	Stats     *StatsHandler
	Debug     *DebugHandler
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
	userPlayerLinksUC *usecase.UserPlayerLinksUseCase,

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

	// blinds
	blindClockUC *usecase.BlindClockService,

	// player
	createPlayer *usecase.CreatePlayerUseCase,
	getPlayers *usecase.GetPlayersUseCase,
	getPlayerStats *usecase.GetPlayerStatsUseCase,

	// stats
	getStatsSessions *usecase.GetStatsSessionsUseCase,
	getStatsPlayers *usecase.GetStatsPlayersUseCase,

	// debug admin
	renameDebugPlayer *usecase.RenameDebugPlayerUseCase,
	updateDebugSessionConfig *usecase.UpdateDebugSessionConfigUseCase,
	deleteDebugPlayer *usecase.DeleteDebugPlayerUseCase,
	deleteDebugSession *usecase.DeleteDebugSessionUseCase,
	deleteDebugSessionFinish *usecase.DeleteDebugSessionFinishUseCase,
) *Handler {

	return &Handler{
		Auth: &AuthHandler{
			authUC:         authUC,
			registerUserUC: registerUserUC,
			cookie:         authCookie,
		},
		Account: &AccountHandler{
			authUC:            authUC,
			userPlayerLinksUC: userPlayerLinksUC,
			cookie:            authCookie,
		},
		Session: &SessionHandler{
			startSessionUC:      startSession,
			finishSessionUC:     finishSession,
			getSessionUC:        getSession,
			getSessionPlayersUC: getSessionPlayers,
			getSessionOpsUC:     getSessionOps,
		},
		Operation: &OperationHandler{
			buyInUC:            buyIn,
			cashOutUC:          cashOut,
			reverseOperationUC: reverse,
		},
		Blinds: &BlindClockHandler{
			service: blindClockUC,
		},
		Player: &PlayerHandler{
			createPlayerUC:   createPlayer,
			getPlayersUC:     getPlayers,
			getPlayerStatsUC: getPlayerStats,
		},
		Stats: &StatsHandler{
			getStatsSessionsUC: getStatsSessions,
			getStatsPlayersUC:  getStatsPlayers,
			authUC:             authUC,
			cookie:             authCookie,
		},
		Debug: &DebugHandler{
			renamePlayerUC:        renameDebugPlayer,
			updateSessionConfigUC: updateDebugSessionConfig,
			deletePlayerUC:        deleteDebugPlayer,
			deleteSessionUC:       deleteDebugSession,
			deleteSessionFinishUC: deleteDebugSessionFinish,
			authUC:                authUC,
			cookie:                authCookie,
		},
	}
}

type AuthHandler struct {
	authUC         *usecase.AuthService
	registerUserUC *usecase.RegisterUserUseCase
	cookie         AuthCookieConfig
}

type AccountHandler struct {
	authUC            *usecase.AuthService
	userPlayerLinksUC *usecase.UserPlayerLinksUseCase
	cookie            AuthCookieConfig
}

type SessionHandler struct {
	startSessionUC      *usecase.StartSessionUseCase
	finishSessionUC     *usecase.FinishSessionUseCase
	getSessionUC        *usecase.GetSessionUseCase
	getSessionPlayersUC *usecase.GetSessionPlayersUseCase
	getSessionOpsUC     *usecase.GetSessionOperationsUseCase
}

type OperationHandler struct {
	buyInUC            *usecase.BuyInUseCase
	cashOutUC          *usecase.CashOutUseCase
	reverseOperationUC *usecase.ReverseOperationUseCase
}

type BlindClockHandler struct {
	service *usecase.BlindClockService
}

type PlayerHandler struct {
	createPlayerUC   *usecase.CreatePlayerUseCase
	getPlayersUC     *usecase.GetPlayersUseCase
	getPlayerStatsUC *usecase.GetPlayerStatsUseCase
}

type StatsHandler struct {
	getStatsSessionsUC *usecase.GetStatsSessionsUseCase
	getStatsPlayersUC  *usecase.GetStatsPlayersUseCase
	authUC             *usecase.AuthService
	cookie             AuthCookieConfig
}

type DebugHandler struct {
	renamePlayerUC        *usecase.RenameDebugPlayerUseCase
	updateSessionConfigUC *usecase.UpdateDebugSessionConfigUseCase
	deletePlayerUC        *usecase.DeleteDebugPlayerUseCase
	deleteSessionUC       *usecase.DeleteDebugSessionUseCase
	deleteSessionFinishUC *usecase.DeleteDebugSessionFinishUseCase
	authUC                *usecase.AuthService
	cookie                AuthCookieConfig
}
