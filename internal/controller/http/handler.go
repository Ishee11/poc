package http

import "github.com/ishee11/poc/internal/usecase"

type Handler struct {
	buyInUC *usecase.BuyInUseCase
}

func NewHandler(buyInUC *usecase.BuyInUseCase) *Handler {
	return &Handler{buyInUC: buyInUC}
}
