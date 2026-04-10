package http

import "github.com/ishee11/poc/internal/usecase"

type Handler struct {
	startSessionUC *usecase.StartSessionUseCase
	buyInUC        *usecase.BuyInUseCase
	getSessionUC   *usecase.GetSessionUseCase
}

func NewHandler(
	startSessionUC *usecase.StartSessionUseCase,
	buyInUC *usecase.BuyInUseCase,
	getSessionUC *usecase.GetSessionUseCase,
) *Handler {
	return &Handler{
		startSessionUC: startSessionUC,
		buyInUC:        buyInUC,
		getSessionUC:   getSessionUC,
	}
}
