package http

import "github.com/ishee11/poc/internal/usecase"

var req struct {
	RequestID string `json:"request_id"`
}

type Handler struct {
	startSessionUC      *usecase.StartSessionUseCase
	buyInUC             *usecase.BuyInUseCase
	cashOutUC           *usecase.CashOutUseCase
	finishSessionUC     *usecase.FinishSessionUseCase
	reverseOperationUC  *usecase.ReverseOperationUseCase
	getSessionUC        *usecase.GetSessionUseCase
	getSessionOpsUC     *usecase.GetSessionOperationsUseCase
	getSessionResultsUC *usecase.GetSessionResultsUseCase
	getStatsSessionsUC  *usecase.GetStatsSessionsUseCase
	getStatsPlayersUC   *usecase.GetStatsPlayersUseCase
	getPlayerStatsUC    *usecase.GetPlayerStatsUseCase
}

func NewHandler(
	start *usecase.StartSessionUseCase,
	buyIn *usecase.BuyInUseCase,
	cashOut *usecase.CashOutUseCase,
	finish *usecase.FinishSessionUseCase,
	reverse *usecase.ReverseOperationUseCase,
	getSession *usecase.GetSessionUseCase,
	getOps *usecase.GetSessionOperationsUseCase,
	getResults *usecase.GetSessionResultsUseCase,
	getStatsSessions *usecase.GetStatsSessionsUseCase,
	getStatsPlayers *usecase.GetStatsPlayersUseCase,
	getPlayerStats *usecase.GetPlayerStatsUseCase,
) *Handler {
	return &Handler{
		startSessionUC:      start,
		buyInUC:             buyIn,
		cashOutUC:           cashOut,
		finishSessionUC:     finish,
		reverseOperationUC:  reverse,
		getSessionUC:        getSession,
		getSessionOpsUC:     getOps,
		getSessionResultsUC: getResults,
		getStatsSessionsUC:  getStatsSessions,
		getStatsPlayersUC:   getStatsPlayers,
		getPlayerStatsUC:    getPlayerStats,
	}
}
