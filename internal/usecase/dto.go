package usecase

import (
	"github.com/ishee11/poc/internal/entity"
)

type PlayerDTO struct {
	ID   entity.PlayerID `json:"player_id"`
	Name string          `json:"name"`
}

type SessionPlayerDTO struct {
	PlayerID entity.PlayerID `json:"player_id"`
	Name     string          `json:"name"`
	Rank     PlayerRank      `json:"rank"`

	BuyIn       int64 `json:"buy_in"`
	CashOut     int64 `json:"cash_out"`
	ProfitChips int64 `json:"profit_chips"`
	ProfitMoney int64 `json:"profit_money"`

	InGame bool `json:"in_game"`
}

type OperationDTO struct {
	ID          entity.OperationID   `json:"id"`
	Type        entity.OperationType `json:"type"`
	PlayerID    entity.PlayerID      `json:"player_id"`
	Chips       int64                `json:"chips"`
	CreatedAt   string               `json:"created_at"`
	ReferenceID *entity.OperationID  `json:"reference_id,omitempty"`
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
	PlayerName     string          `json:"player_name"`
	SessionsCount  int64           `json:"sessions_count"`
	TotalBuyIn     int64           `json:"total_buy_in"`
	TotalCashOut   int64           `json:"total_cash_out"`
	ProfitChips    int64           `json:"profit_chips"`
	ProfitMoney    int64           `json:"profit_money"`
	LastActivityAt *string         `json:"last_activity_at"`
	Rank           PlayerRank      `json:"rank"`
}

type PlayerRank struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type SessionStat struct {
	SessionID    entity.SessionID `json:"session_id"`
	Status       entity.Status    `json:"status"`
	ChipRate     int64            `json:"chip_rate"`
	BigBlind     int64            `json:"big_blind"`
	Currency     entity.Currency  `json:"currency"`
	CreatedAt    string           `json:"created_at"`
	FinishedAt   *string          `json:"finished_at,omitempty"`
	TotalBuyIn   int64            `json:"total_buy_in"`
	TotalCashOut int64            `json:"total_cash_out"`
	PlayerCount  int64            `json:"player_count"`
}

type PlayerOverallStat struct {
	PlayerID            entity.PlayerID      `json:"player_id"`
	PlayerName          string               `json:"player_name"`
	SessionsCount       int64                `json:"sessions_count"`
	TotalBuyIn          int64                `json:"total_buy_in"`
	TotalCashOut        int64                `json:"total_cash_out"`
	TotalBuyInMoney     int64                `json:"total_buy_in_money"`
	TotalCashOutMoney   int64                `json:"total_cash_out_money"`
	ProfitChips         int64                `json:"profit_chips"`
	ProfitMoney         int64                `json:"profit_money"`
	AvgProfitPerSession float64              `json:"avg_profit_per_session"`
	ROIPercent          float64              `json:"roi_percent"`
	AvgBuyInPerSession  float64              `json:"avg_buy_in_per_session"`
	LastActivityAt      *string              `json:"last_activity_at"`
	Rank                PlayerRank           `json:"rank"`
	MoneyByCurrency     []PlayerCurrencyStat `json:"money_by_currency"`
}

type PlayerCurrencyStat struct {
	Currency            entity.Currency `json:"currency"`
	SessionsCount       int64           `json:"sessions_count"`
	TotalBuyInMoney     int64           `json:"total_buy_in_money"`
	TotalCashOutMoney   int64           `json:"total_cash_out_money"`
	ProfitMoney         int64           `json:"profit_money"`
	AvgProfitPerSession float64         `json:"avg_profit_per_session"`
	ROIPercent          float64         `json:"roi_percent"`
}
