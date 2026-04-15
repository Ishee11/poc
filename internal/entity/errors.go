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
	ErrInsufficientTableChips = errors.New("not enough chips")

	ErrPlayersStillInGame   = errors.New("players still have chips in game")
	ErrInvalidPlayerID      = errors.New("invalid player id")
	ErrInvalidPlayerName    = errors.New("invalid player name")
	ErrSessionNotFinished   = errors.New("session is not finished")
	ErrUnbalancedSession    = errors.New("session money is not balanced")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrInvalidMoney         = errors.New("invalid money")

	ErrInvalidOperationType     = errors.New("invalid operation type")
	ErrInvalidReference         = errors.New("invalid reference")
	ErrOperationNotFound        = errors.New("operation not found")
	ErrOperationAlreadyReversed = errors.New("operation already reversed")
	ErrInvalidRequestID         = errors.New("invalid request id")
	ErrDuplicateRequest         = errors.New("duplicate request")

	ErrPlayerNotInGame  = errors.New("player not in game")
	ErrInvalidOperation = errors.New("invalid operation")
	ErrInvalidCashOut   = errors.New("invalid cash out")
	ErrTableNotSettled  = errors.New("table not settled")

	ErrSessionNotBalanced = errors.New("session not balanced")
)

func (e *SessionNotBalancedError) Is(target error) bool {
	return target == ErrSessionNotBalanced
}

type SessionNotBalancedError struct {
	RemainingChips int64
}

func (e *SessionNotBalancedError) Error() string {
	return "session not balanced"
}
