package command

import "github.com/ishee11/poc/internal/entity"

type StartSessionCommand struct {
	SessionID entity.SessionID
	ChipRate  int64
}

type FinishSessionCommand struct {
	RequestID string
	SessionID entity.SessionID
}
