package usecase

import "github.com/ishee11/poc/internal/entity"

// --- OperationRepository mock ---

type operationRepoMock struct {
	saveFn func(tx Tx, op *entity.Operation) error

	getLastOpFn func(tx Tx, sessionID entity.SessionID, playerID entity.PlayerID) (entity.OperationType, bool, error)
	getAggFn    func(tx Tx, sessionID entity.SessionID) (SessionAggregates, error)

	getByIDFn        func(tx Tx, id entity.OperationID) (*entity.Operation, error)
	existsReversalFn func(tx Tx, targetID entity.OperationID) (bool, error)

	getByRequestIDFn func(tx Tx, requestID string) (*entity.Operation, error)
}

func (m *operationRepoMock) Save(tx Tx, op *entity.Operation) error {
	if m.saveFn != nil {
		return m.saveFn(tx, op)
	}
	return nil
}

func (m *operationRepoMock) GetLastOperationType(
	tx Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) (entity.OperationType, bool, error) {
	if m.getLastOpFn != nil {
		return m.getLastOpFn(tx, sessionID, playerID)
	}
	return "", false, nil
}

func (m *operationRepoMock) GetSessionAggregates(
	tx Tx,
	sessionID entity.SessionID,
) (SessionAggregates, error) {
	if m.getAggFn != nil {
		return m.getAggFn(tx, sessionID)
	}
	return SessionAggregates{}, nil
}

func (m *operationRepoMock) GetByID(tx Tx, id entity.OperationID) (*entity.Operation, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(tx, id)
	}
	return nil, nil
}

func (m *operationRepoMock) ExistsReversal(tx Tx, targetID entity.OperationID) (bool, error) {
	if m.existsReversalFn != nil {
		return m.existsReversalFn(tx, targetID)
	}
	return false, nil
}

func (m *operationRepoMock) GetByRequestID(tx Tx, requestID string) (*entity.Operation, error) {
	if m.getByRequestIDFn != nil {
		return m.getByRequestIDFn(tx, requestID)
	}
	return nil, nil
}

// --- SessionRepository mock ---

type sessionRepoMock struct {
	findFn func(tx Tx, id entity.SessionID) (*entity.Session, error)
	saveFn func(tx Tx, s *entity.Session) error
}

func (m *sessionRepoMock) FindByID(tx Tx, id entity.SessionID) (*entity.Session, error) {
	if m.findFn != nil {
		return m.findFn(tx, id)
	}
	return nil, nil
}

func (m *sessionRepoMock) Save(tx Tx, s *entity.Session) error {
	if m.saveFn != nil {
		return m.saveFn(tx, s)
	}
	return nil
}

// --- TxManager mock ---

type txManagerMock struct{}

func (m *txManagerMock) RunInTx(fn func(tx Tx) error) error {
	return fn(struct{}{})
}

// --- OperationIDGenerator mock ---

type operationIDGeneratorMock struct {
	id entity.OperationID
}

func (m *operationIDGeneratorMock) New() entity.OperationID {
	return m.id
}
