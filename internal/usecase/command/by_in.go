package command

import "github.com/ishee11/poc/internal/entity"

type BuyInCommand struct {
	RequestID string
	SessionID entity.SessionID
	PlayerID  entity.PlayerID
	Chips     int64
}
