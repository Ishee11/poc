package http

import "github.com/ishee11/poc/internal/entity"

type StartSessionRequest struct {
	ChipRate int64 `json:"chip_rate"`
	BigBlind int64 `json:"big_blind"`
}

type FinishSessionRequest struct {
	RequestID string `json:"request_id"`
	SessionID string `json:"session_id"`
}

type ErrorResponse struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
}

type ReverseOperationRequest struct {
	RequestID         string `json:"request_id" example:"req-123"`
	TargetOperationID string `json:"target_operation_id" example:"op-456"`
}

type BuyInRequest struct {
	RequestID string `json:"request_id" example:"req-123"`
	SessionID string `json:"session_id" example:"session-1"`
	PlayerID  string `json:"player_id" example:"player-1"`
	Chips     int64  `json:"chips" example:"1000"`
}

type CashOutRequest struct {
	RequestID string `json:"request_id" example:"req-123"`
	SessionID string `json:"session_id" example:"session-1"`
	PlayerID  string `json:"player_id" example:"player-1"`
	Chips     int64  `json:"chips" example:"500"`
}

type CreatePlayerRequest struct {
	RequestID string `json:"request_id" example:"req-123"`
	Name      string `json:"name" example:"Alice"`
}

type RenamePlayerRequest struct {
	Name string `json:"name" example:"Alice"`
}

type CreatePlayerResponse struct {
	PlayerID entity.PlayerID `json:"player_id" example:"player-123"`
}
