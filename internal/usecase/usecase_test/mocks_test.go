package usecase_test

import (
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// --- OperationRepository mock ---

type operationRepoMock struct {
	saveFn func(tx usecase.Tx, op *entity.Operation) error

	getLastOpFn func(tx usecase.Tx, sessionID entity.SessionID, playerID entity.PlayerID) (entity.OperationType, bool, error)
	getAggFn    func(tx usecase.Tx, sessionID entity.SessionID) (usecase.SessionAggregates, error)

	getByIDFn        func(tx usecase.Tx, id entity.OperationID) (*entity.Operation, error)
	existsReversalFn func(tx usecase.Tx, targetID entity.OperationID) (bool, error)

	getByRequestIDFn func(tx usecase.Tx, requestID string) (*entity.Operation, error)

	listBySessionFn func(tx usecase.Tx, sessionID entity.SessionID, limit int, offset int) ([]*entity.Operation, error)
	getPlayerAggFn  func(tx usecase.Tx, sessionID entity.SessionID) (map[entity.PlayerID]usecase.PlayerAggregates, error)
}

func (m *operationRepoMock) Save(tx usecase.Tx, op *entity.Operation) error {
	if m.saveFn != nil {
		return m.saveFn(tx, op)
	}
	return nil
}

func (m *operationRepoMock) ListBySession(
	tx usecase.Tx,
	sessionID entity.SessionID,
	limit int,
	offset int,
) ([]*entity.Operation, error) {

	if m.listBySessionFn != nil {
		return m.listBySessionFn(tx, sessionID, limit, offset)
	}

	return []*entity.Operation{}, nil
}

func (m *operationRepoMock) GetLastOperationType(
	tx usecase.Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) (entity.OperationType, bool, error) {
	if m.getLastOpFn != nil {
		return m.getLastOpFn(tx, sessionID, playerID)
	}
	return "", false, nil
}

func (m *operationRepoMock) GetSessionAggregates(
	tx usecase.Tx,
	sessionID entity.SessionID,
) (usecase.SessionAggregates, error) {
	if m.getAggFn != nil {
		return m.getAggFn(tx, sessionID)
	}
	return usecase.SessionAggregates{}, nil
}

func (m *operationRepoMock) GetByID(tx usecase.Tx, id entity.OperationID) (*entity.Operation, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(tx, id)
	}
	return nil, nil
}

func (m *operationRepoMock) ExistsReversal(tx usecase.Tx, targetID entity.OperationID) (bool, error) {
	if m.existsReversalFn != nil {
		return m.existsReversalFn(tx, targetID)
	}
	return false, nil
}

func (m *operationRepoMock) GetByRequestID(tx usecase.Tx, requestID string) (*entity.Operation, error) {
	if m.getByRequestIDFn != nil {
		return m.getByRequestIDFn(tx, requestID)
	}
	return nil, nil
}

func (m *operationRepoMock) GetPlayerAggregates(
	tx usecase.Tx,
	sessionID entity.SessionID,
) (map[entity.PlayerID]usecase.PlayerAggregates, error) {
	if m.getPlayerAggFn != nil {
		return m.getPlayerAggFn(tx, sessionID)
	}
	return map[entity.PlayerID]usecase.PlayerAggregates{}, nil
}

// --- SessionRepository mock ---

type sessionRepoMock struct {
	findFn func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error)
	saveFn func(tx usecase.Tx, s *entity.Session) error
}

func (m *sessionRepoMock) FindByID(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
	if m.findFn != nil {
		return m.findFn(tx, id)
	}
	return nil, nil
}

func (m *sessionRepoMock) Save(tx usecase.Tx, s *entity.Session) error {
	if m.saveFn != nil {
		return m.saveFn(tx, s)
	}
	return nil
}

// --- TxManager mock ---

type txManagerMock struct{}

func (m *txManagerMock) RunInTx(fn func(tx usecase.Tx) error) error {
	return fn(struct{}{})
}

// --- OperationIDGenerator mock ---

type operationIDGeneratorMock struct {
	id entity.OperationID
}

func (m *operationIDGeneratorMock) New() entity.OperationID {
	return m.id
}

// --- IdempotencyRepository mock ---

type idempotencyRepoMock struct {
	saveFn func(tx usecase.Tx, requestID string) error
}

func (m *idempotencyRepoMock) Save(tx usecase.Tx, requestID string) error {
	if m.saveFn != nil {
		return m.saveFn(tx, requestID)
	}
	return nil
}

// --- PlayerRepository mock ---

type playerRepoMock struct{}

func (m *playerRepoMock) GetOrCreate(
	tx usecase.Tx,
	sessionID entity.SessionID,
	name string,
) (entity.PlayerID, error) {
	return entity.PlayerID(name), nil
}

func (m *playerRepoMock) ListBySession(
	tx usecase.Tx,
	sessionID entity.SessionID,
) ([]usecase.PlayerDTO, error) {
	return []usecase.PlayerDTO{}, nil
}
