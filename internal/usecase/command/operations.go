package command

import "github.com/ishee11/poc/internal/entity"

type ReverseOperationCommand struct {
	RequestID         string
	SessionID         entity.SessionID
	TargetOperationID entity.OperationID
}
