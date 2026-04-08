package usecase

import (
	"sync"

	"github.com/ishee11/poc/internal/entity"
)

type inMemoryOperationRepo struct {
	mu         sync.Mutex
	operations []*entity.Operation
}

func (r *inMemoryOperationRepo) Save(tx Tx, op *entity.Operation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.operations {
		if existing.ID() == op.ID() {
			return entity.ErrDuplicateOperation
		}
	}

	r.operations = append(r.operations, op)
	return nil
}

func (r *inMemoryOperationRepo) GetLastOperationType(
	tx Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) (entity.OperationType, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := len(r.operations) - 1; i >= 0; i-- {
		op := r.operations[i]
		if op.SessionID() == sessionID && op.PlayerID() == playerID {
			return op.Type(), true, nil
		}
	}

	return "", false, nil
}

func (r *inMemoryOperationRepo) GetSessionAggregates(
	tx Tx,
	sessionID entity.SessionID,
) (SessionAggregates, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var buyIn, cashOut int64

	for _, op := range r.operations {
		if op.SessionID() != sessionID {
			continue
		}

		switch op.Type() {
		case entity.OperationBuyIn:
			buyIn += op.Chips()
		case entity.OperationCashOut:
			cashOut += op.Chips()
		}
	}

	return SessionAggregates{
		TotalBuyIn:   buyIn,
		TotalCashOut: cashOut,
	}, nil
}

type inMemorySessionRepo struct {
	mu       sync.Mutex
	sessions map[entity.SessionID]*entity.Session
}

func newSessionRepo() *inMemorySessionRepo {
	return &inMemorySessionRepo{
		sessions: make(map[entity.SessionID]*entity.Session),
	}
}

func (r *inMemorySessionRepo) FindByID(tx Tx, id entity.SessionID) (*entity.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.sessions[id], nil
}

func (r *inMemorySessionRepo) Save(tx Tx, s *entity.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[s.ID()] = s
	return nil
}

type txManagerStub struct{}

func (m *txManagerStub) RunInTx(fn func(tx Tx) error) error {
	return fn(struct{}{})
}
