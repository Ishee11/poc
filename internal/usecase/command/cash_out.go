package command

import "github.com/ishee11/poc/internal/entity"

type CashOutCommand struct {
	RequestID  string
	SessionID  entity.SessionID
	PlayerName string
	Chips      int64
}
