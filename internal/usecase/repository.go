package usecase

import "github.com/ishee11/poc/internal/entity"

type SessionAggregates struct {
	TotalBuyIn   int64
	TotalCashOut int64
}

type OperationWriter interface {
	Save(tx Tx, op *entity.Operation) error
}

type OperationReader interface {
	GetByID(tx Tx, id entity.OperationID) (*entity.Operation, error)
	GetByRequestID(tx Tx, requestID string) (*entity.Operation, error)
}

type OperationAggregateReader interface {
	GetSessionAggregates(tx Tx, sessionID entity.SessionID) (SessionAggregates, error)
}

type OperationPlayerStateReader interface {
	GetLastOperationType(tx Tx, sessionID entity.SessionID, playerID entity.PlayerID) (entity.OperationType, bool, error)
}

type OperationReversalChecker interface {
	ExistsReversal(tx Tx, targetID entity.OperationID) (bool, error)
}

type SessionReader interface {
	FindByID(tx Tx, sessionID entity.SessionID) (*entity.Session, error)
}

type SessionWriter interface {
	Save(tx Tx, session *entity.Session) error
}
