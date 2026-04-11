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
	}
}
