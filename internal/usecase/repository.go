package usecase

import "github.com/ishee11/poc/internal/entity"

type SessionAggregates struct {
	TotalBuyIn   int64
	TotalCashOut int64
}

type OperationRepository interface {
	Save(tx Tx, op *entity.Operation) error
	GetLastOperationType(tx Tx, sessionID entity.SessionID, playerID entity.PlayerID) (entity.OperationType, bool, error)
	GetSessionAggregates(tx Tx, sessionID entity.SessionID) (SessionAggregates, error)
	ExistsReversal(tx Tx, targetID entity.OperationID) (bool, error)
	GetByID(tx Tx, id entity.OperationID) (*entity.Operation, error)
	GetByRequestID(tx Tx, requestID string) (*entity.Operation, error)
}
type SessionRepository interface {
	FindByID(tx Tx, sessionID entity.SessionID) (*entity.Session, error)
	Save(tx Tx, session *entity.Session) error
}
