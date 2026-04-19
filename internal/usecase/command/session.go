package command

import "github.com/ishee11/poc/internal/entity"

type StartSessionCommand struct {
	ChipRate int64
	BigBlind int64
}

type FinishSessionCommand struct {
	RequestID string
	SessionID entity.SessionID
}
