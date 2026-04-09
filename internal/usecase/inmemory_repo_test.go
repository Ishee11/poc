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
		// защита по requestID (идемпотентность)
		if existing.RequestID() == op.RequestID() {
			return entity.ErrDuplicateRequest
		}
		// защита по ID (техническая)
		if existing.ID() == op.ID() {
			return nil
		}
	}

	r.operations = append(r.operations, op)
	return nil
}

func (r *inMemoryOperationRepo) GetByRequestID(tx Tx, requestID string) (*entity.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, op := range r.operations {
		if op.RequestID() == requestID {
			return op, nil
		}
	}
	return nil, nil
}

func (r *inMemoryOperationRepo) GetLastOperationType(
	tx Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) (entity.OperationType, bool, error) {

	r.mu.Lock()
	defer r.mu.Unlock()

	reversed := make(map[entity.OperationID]struct{})

	for _, op := range r.operations {
		if op.Type() == entity.OperationReversal && op.ReferenceID() != nil {
			reversed[*op.ReferenceID()] = struct{}{}
		}
	}

	for i := len(r.operations) - 1; i >= 0; i-- {
		op := r.operations[i]

		if op.SessionID() != sessionID || op.PlayerID() != playerID {
			continue
		}

		if op.Type() == entity.OperationReversal {
			continue
		}

		if _, ok := reversed[op.ID()]; ok {
			continue
		}

		return op.Type(), true, nil
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

		case entity.OperationReversal:
			refID := op.ReferenceID()
			if refID == nil {
				continue
			}

			target := r.findByID(*refID)
			if target == nil {
				continue
			}

			switch target.Type() {
			case entity.OperationBuyIn:
				buyIn -= target.Chips()
			case entity.OperationCashOut:
				cashOut -= target.Chips()
			}
		}
	}

	return SessionAggregates{
		TotalBuyIn:   buyIn,
		TotalCashOut: cashOut,
	}, nil
}

func (r *inMemoryOperationRepo) GetByID(tx Tx, id entity.OperationID) (*entity.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, op := range r.operations {
		if op.ID() == id {
			return op, nil
		}
	}
	return nil, nil
}

func (r *inMemoryOperationRepo) ExistsReversal(tx Tx, targetID entity.OperationID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, op := range r.operations {
		if op.Type() == entity.OperationReversal &&
			op.ReferenceID() != nil &&
			*op.ReferenceID() == targetID {
			return true, nil
		}
	}
	return false, nil
}

// внутренний helper (БЕЗ lock)
func (r *inMemoryOperationRepo) findByID(id entity.OperationID) *entity.Operation {
	for _, op := range r.operations {
		if op.ID() == id {
			return op
		}
	}
	return nil
}

// --- Session repo ---

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

// --- Tx stub ---

type txManagerStub struct{}

func (m *txManagerStub) RunInTx(fn func(tx Tx) error) error {
	return fn(struct{}{})
}
