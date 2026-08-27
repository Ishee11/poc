package command

import "github.com/ishee11/poc/internal/entity"

type StartSessionCommand struct {
	UserID   entity.AuthUserID
	ChipRate int64
	BigBlind int64
	Currency entity.Currency
}

type FinishSessionCommand struct {
	RequestID string
	SessionID entity.SessionID
}
