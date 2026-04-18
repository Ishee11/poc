package http

import "github.com/ishee11/poc/internal/usecase"

type Handler struct {
	Session   *SessionHandler
	Operation *OperationHandler
	Player    *PlayerHandler
	Stats     *StatsHandler
	Debug     *DebugHandler
}

func NewHandler(
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

	// player
	createPlayer *usecase.CreatePlayerUseCase,
	getPlayers *usecase.GetPlayersUseCase,
	getPlayerStats *usecase.GetPlayerStatsUseCase,

	// stats
	getStatsSessions *usecase.GetStatsSessionsUseCase,
	getStatsPlayers *usecase.GetStatsPlayersUseCase,

	// debug admin
	deleteDebugPlayer *usecase.DeleteDebugPlayerUseCase,
	deleteDebugSession *usecase.DeleteDebugSessionUseCase,
) *Handler {

	return &Handler{
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
		Player: &PlayerHandler{
			createPlayerUC:   createPlayer,
			getPlayersUC:     getPlayers,
			getPlayerStatsUC: getPlayerStats,
		},
		Stats: &StatsHandler{
			getStatsSessionsUC: getStatsSessions,
			getStatsPlayersUC:  getStatsPlayers,
		},
		Debug: &DebugHandler{
			deletePlayerUC:  deleteDebugPlayer,
			deleteSessionUC: deleteDebugSession,
		},
	}
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

type PlayerHandler struct {
	createPlayerUC   *usecase.CreatePlayerUseCase
	getPlayersUC     *usecase.GetPlayersUseCase
	getPlayerStatsUC *usecase.GetPlayerStatsUseCase
}

type StatsHandler struct {
	getStatsSessionsUC *usecase.GetStatsSessionsUseCase
	getStatsPlayersUC  *usecase.GetStatsPlayersUseCase
}

type DebugHandler struct {
	deletePlayerUC  *usecase.DeleteDebugPlayerUseCase
	deleteSessionUC *usecase.DeleteDebugSessionUseCase
}
