package usecase

import "github.com/ishee11/poc/internal/entity"

type OperationRepository interface {
	Save(tx Tx, op *entity.Operation) error
}
type SessionRepository interface {
	FindByID(tx Tx, sessionID entity.SessionID) (*entity.Session, error)
	Save(tx Tx, session *entity.Session) error
}
