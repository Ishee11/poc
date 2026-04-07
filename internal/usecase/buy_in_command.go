package usecase

import "github.com/ishee11/poc/internal/entity"

type BuyInCommand struct {
	OperationID entity.OperationID
	SessionID   entity.SessionID
	PlayerID    entity.PlayerID
	Chips       int64
}
