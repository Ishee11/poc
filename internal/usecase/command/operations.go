package command

import "github.com/ishee11/poc/internal/entity"

type ReverseOperationCommand struct {
	RequestID         string
	TargetOperationID entity.OperationID
}
