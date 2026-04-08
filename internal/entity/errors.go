package entity

import "errors"

var (
	ErrSessionNotActive      = errors.New("session not active")
	ErrSessionNotCreated     = errors.New("session not created")
	ErrSessionFinished       = errors.New("session finished")
	ErrNotEnoughChipsOnTable = errors.New("not enough chips on table")
	ErrInvalidChipAmount     = errors.New("invalid chip amount")

	ErrInvalidChips           = errors.New("chips must be greater than 0")
	ErrPlayerNotFound         = errors.New("player not found")
	ErrPlayerStillInGame      = errors.New("player still in game")
	ErrTableNotSettled        = errors.New("table not settled")
	ErrInsufficientTableChips = errors.New("not enough chips")

	ErrPlayersStillInGame   = errors.New("players still have chips in game")
	ErrSessionNotFinished   = errors.New("session is not finished")
	ErrUnbalancedSession    = errors.New("session money is not balanced")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrInvalidMoney         = errors.New("invalid money")

	ErrInvalidOperationType = errors.New("invalid operation type")

	ErrDuplicateOperation = errors.New("duplicate operation")
	ErrPlayerNotInGame    = errors.New("player not in game")
	ErrInvalidOperation   = errors.New("invalid operation")
	ErrInvalidCashOut     = errors.New("invalid cash out")
)
