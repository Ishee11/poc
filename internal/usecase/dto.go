package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type PlayerDTO struct {
	ID   entity.PlayerID
	Name string
}

type OperationDTO struct {
	ID          entity.OperationID
	Type        entity.OperationType
	PlayerID    entity.PlayerID
	Chips       int64
	CreatedAt   time.Time
	ReferenceID *entity.OperationID
}

type PlayerResultDTO struct {
	PlayerID     entity.PlayerID
	BuyInChips   int64
	CashOutChips int64
	ProfitChips  int64
	ProfitMoney  int64
}
