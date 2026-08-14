package usecase

import (
	"errors"

	"github.com/ishee11/poc/internal/entity"
)

type IdempotencyRepository interface {
	Save(tx Tx, requestID string) error
}

type OperationFingerprint struct {
	Type              entity.OperationType
	SessionID         entity.SessionID
	PlayerID          entity.PlayerID
	Chips             int64
	TargetOperationID entity.OperationID
}

func (f OperationFingerprint) matches(op *entity.Operation) bool {
	if op == nil || op.Type() != f.Type || op.SessionID() != f.SessionID ||
		op.PlayerID() != f.PlayerID || op.Chips() != f.Chips {
		return false
	}
	if f.Type != entity.OperationReversal {
		return op.ReferenceID() == nil
	}
	return op.ReferenceID() != nil && *op.ReferenceID() == f.TargetOperationID
}

func IdempotentOperation(
	tx Tx,
	repo IdempotencyRepository,
	reader OperationReader,
	requestID string,
	fingerprint OperationFingerprint,
	fn func() (*entity.Operation, error),
) (*entity.Operation, bool, error) {
	if requestID == "" {
		return nil, false, entity.ErrInvalidRequestID
	}
	if err := repo.Save(tx, requestID); err != nil {
		if !errors.Is(err, entity.ErrDuplicateRequest) {
			return nil, false, err
		}
		op, loadErr := reader.GetByRequestID(tx, requestID)
		if loadErr != nil || !fingerprint.matches(op) {
			return nil, false, entity.ErrIdempotencyPayloadMismatch
		}
		return op, true, nil
	}
	if _, err := fn(); err != nil {
		return nil, false, err
	}
	persisted, err := reader.GetByRequestID(tx, requestID)
	if err != nil || !fingerprint.matches(persisted) {
		return nil, false, entity.ErrIdempotencyPayloadMismatch
	}
	return persisted, false, nil
}

func Idempotent(
	tx Tx,
	repo IdempotencyRepository,
	requestID string,
	fn func() error,
) error {
	if requestID == "" {
		return entity.ErrInvalidRequestID
	}

	// 1. пробуем записать request_id
	if err := repo.Save(tx, requestID); err != nil {
		if errors.Is(err, entity.ErrDuplicateRequest) {
			return nil
		}
		return err
	}

	// 2. выполняем бизнес-логику
	return fn()
}
