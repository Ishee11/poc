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

type PlayerStat struct {
	PlayerID       entity.PlayerID `json:"player_id"`
	SessionsCount  int64           `json:"sessions_count"`
	TotalBuyIn     int64           `json:"total_buy_in"`
	TotalCashOut   int64           `json:"total_cash_out"`
	ProfitChips    int64           `json:"profit_chips"`
	ProfitMoney    int64           `json:"profit_money"`
	LastActivityAt *string         `json:"last_activity_at"`
}

type SessionStat struct {
	SessionID    entity.SessionID `json:"session_id"`
	Status       entity.Status    `json:"status"`
	ChipRate     int64            `json:"chip_rate"`
	CreatedAt    string           `json:"created_at"`
	TotalBuyIn   int64            `json:"total_buy_in"`
	TotalCashOut int64            `json:"total_cash_out"`
	PlayerCount  int64            `json:"player_count"`
}

type PlayerOverallStat struct {
	PlayerID       entity.PlayerID `json:"player_id"`
	SessionsCount  int64           `json:"sessions_count"`
	TotalBuyIn     int64           `json:"total_buy_in"`
	TotalCashOut   int64           `json:"total_cash_out"`
	ProfitChips    int64           `json:"profit_chips"`
	ProfitMoney    int64           `json:"profit_money"`
	LastActivityAt *string         `json:"last_activity_at"`
}
