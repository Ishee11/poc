package postgres

import "errors"

var (
	ErrInvalidTx = errors.New("invalid transaction")
)
