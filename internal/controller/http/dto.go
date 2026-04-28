package http

import (
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type StartSessionRequest struct {
	ChipRate int64  `json:"chip_rate"`
	BigBlind int64  `json:"big_blind"`
	Currency string `json:"currency"`
}

type FinishSessionRequest struct {
	RequestID string `json:"request_id"`
	SessionID string `json:"session_id"`
}

type ErrorResponse struct {
	Error     string      `json:"error"`
	RequestID string      `json:"request_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
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

type UpdateSessionConfigRequest struct {
	ChipRate int64  `json:"chip_rate"`
	BigBlind int64  `json:"big_blind"`
	Currency string `json:"currency"`
}

type CreatePlayerResponse struct {
	PlayerID entity.PlayerID `json:"player_id" example:"player-123"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthUserResponse struct {
	ID    entity.AuthUserID `json:"id"`
	Email string            `json:"email"`
	Role  entity.AuthRole   `json:"role"`
}

type LoginResponse struct {
	User      AuthUserResponse `json:"user"`
	ExpiresAt string           `json:"expires_at"`
}

type MeResponse struct {
	User AuthUserResponse `json:"user"`
}

type AccountResponse struct {
	User    AuthUserResponse `json:"user"`
	Players []PlayerDTO      `json:"players"`
}

type PlayerDTO struct {
	ID   entity.PlayerID `json:"player_id"`
	Name string          `json:"name"`
}

type AccountPlayersResponse struct {
	Players []PlayerDTO `json:"players"`
}

type LinkAccountPlayerRequest struct {
	PlayerID string `json:"player_id"`
}

type UpdateBlindClockLevelsRequest struct {
	Levels []BlindClockLevelRequest `json:"levels"`
}

type BlindClockLevelRequest struct {
	SmallBlind      int64 `json:"small_blind"`
	BigBlind        int64 `json:"big_blind"`
	DurationMinutes int64 `json:"duration_minutes"`
}

type PushSubscribeKeysRequest struct {
	Auth   string `json:"auth"`
	P256DH string `json:"p256dh"`
}

type PushSubscribeRequest struct {
	Endpoint  string                   `json:"endpoint"`
	Keys      PushSubscribeKeysRequest `json:"keys"`
	UserAgent string                   `json:"user_agent"`
}

func (r PushSubscribeRequest) toInput() usecase.BlindClockPushSubscriptionInput {
	return usecase.BlindClockPushSubscriptionInput{
		Endpoint:  r.Endpoint,
		KeyAuth:   r.Keys.Auth,
		KeyP256DH: r.Keys.P256DH,
		UserAgent: r.UserAgent,
	}
}

type PushUnsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}
