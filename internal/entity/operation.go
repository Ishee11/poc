package entity

import (
	"time"
)

type OperationType string
type OperationID string
type PlayerID string

const (
	OperationBuyIn    OperationType = "buy_in"
	OperationCashOut  OperationType = "cash_out"
	OperationReversal OperationType = "reversal"
)

type Operation struct {
	id            OperationID
	sessionID     SessionID
	operationType OperationType
	playerID      PlayerID
	chips         int64
	createdAt     time.Time
	referenceID   *OperationID
}

func NewOperation(
	id OperationID,
	sessionID SessionID,
	operationType OperationType,
	playerID PlayerID,
	chips int64,
	date time.Time,
) (*Operation, error) {

	if chips <= 0 {
		return nil, ErrInvalidChips
	}

	if operationType != OperationBuyIn && operationType != OperationCashOut {
		return nil, ErrInvalidOperationType
	}

	return &Operation{
		id:            id,
		sessionID:     sessionID,
		operationType: operationType,
		playerID:      playerID,
		chips:         chips,
		createdAt:     date,
	}, nil
}

func NewReversalOperation(
	id OperationID,
	sessionID SessionID,
	playerID PlayerID,
	chips int64,
	referenceID OperationID,
	date time.Time,
) (*Operation, error) {

	if chips <= 0 {
		return nil, ErrInvalidChips
	}

	if referenceID == "" {
		return nil, ErrInvalidReference
	}

	return &Operation{
		id:            id,
		sessionID:     sessionID,
		operationType: OperationReversal,
		playerID:      playerID,
		chips:         chips,
		createdAt:     date,
		referenceID:   &referenceID,
	}, nil
}

func (o *Operation) ID() OperationID {
	return o.id
}

func (o *Operation) SessionID() SessionID {
	return o.sessionID
}

func (o *Operation) PlayerID() PlayerID {
	return o.playerID
}

func (o *Operation) Type() OperationType {
	return o.operationType
}

func (o *Operation) Chips() int64 {
	return o.chips
}

func (o *Operation) ReferenceID() *OperationID {
	return o.referenceID
}
