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
	PlayerID     entity.PlayerID `json:"player_id"`
	PlayerName   string          `json:"player_name"`
	BuyInChips   int64           `json:"buy_in_chips"`
	CashOutChips int64           `json:"cash_out_chips"`
	ProfitChips  int64           `json:"profit_chips"`
	ProfitMoney  int64           `json:"profit_money"`
}
