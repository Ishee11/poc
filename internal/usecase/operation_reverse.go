// NOTE: No DB-level protection for reversal uniqueness (acceptable for single-user scenario).
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type ReverseOperationUseCase struct {
	opWriter        OperationWriter
	opReader        OperationReader
	reversalChecker OperationReversalChecker
	sessionWriter   SessionWriter
	txManager       TxManager
	idGen           OperationIDGenerator
	idempotencyRepo IdempotencyRepository
	sessionLocker   SessionLocker
	outboxWriter    OutboxWriter
}

func NewReverseOperationUseCase(
	opWriter OperationWriter,
	opReader OperationReader,
	reversalChecker OperationReversalChecker,
	sessionWriter SessionWriter,
	txManager TxManager,
	idGen OperationIDGenerator,
	idempotencyRepo IdempotencyRepository,
	sessionLocker SessionLocker,
	outboxWriter OutboxWriter,
) *ReverseOperationUseCase {
	return &ReverseOperationUseCase{
		opWriter:        opWriter,
		opReader:        opReader,
		reversalChecker: reversalChecker,
		sessionWriter:   sessionWriter,
		txManager:       txManager,
		idGen:           idGen,
		idempotencyRepo: idempotencyRepo,
		sessionLocker:   sessionLocker,
		outboxWriter:    outboxWriter,
	}
}

func (uc *ReverseOperationUseCase) Execute(ctx context.Context, cmd command.ReverseOperationCommand) (OperationAcknowledgement, error) {
	var acknowledgement OperationAcknowledgement
	err := uc.txManager.RunInTx(ctx, func(tx Tx) error {
		if cmd.RequestID == "" {
			return entity.ErrInvalidRequestID
		}
		duplicate := false
		var op *entity.Operation
		if err := uc.idempotencyRepo.Save(tx, cmd.RequestID); err != nil {
			if !errors.Is(err, entity.ErrDuplicateRequest) {
				return err
			}
			duplicate = true
			op, err = uc.opReader.GetByRequestID(tx, cmd.RequestID)
			if err != nil || op == nil || op.Type() != entity.OperationReversal || op.ReferenceID() == nil || *op.ReferenceID() != cmd.TargetOperationID {
				return entity.ErrIdempotencyPayloadMismatch
			}
		} else {
			target, err := uc.opReader.GetByID(tx, cmd.TargetOperationID)
			if err != nil {
				return err
			}
			if _, err = uc.execute(tx, cmd, target); err != nil {
				return err
			}
			op, err = uc.opReader.GetByRequestID(tx, cmd.RequestID)
			if err != nil || op == nil || op.Type() != entity.OperationReversal || op.ReferenceID() == nil || *op.ReferenceID() != cmd.TargetOperationID {
				return entity.ErrIdempotencyPayloadMismatch
			}
		}
		target, err := uc.opReader.GetByID(tx, *op.ReferenceID())
		if err != nil {
			return err
		}
		acknowledgement = NewOperationAcknowledgement(op, duplicate)
		acknowledgement.TargetOperationID = op.ReferenceID()
		reversed := NewPersistedOperationDetails(target)
		acknowledgement.ReversedOperation = &reversed
		return nil
	})
	return acknowledgement, err
}

func (uc *ReverseOperationUseCase) execute(tx Tx, cmd command.ReverseOperationCommand, target *entity.Operation) (*entity.Operation, error) {
	// 1. target operation
	if target == nil {
		return nil, entity.ErrOperationNotFound
	}

	if target.Type() == entity.OperationReversal {
		return nil, entity.ErrInvalidOperation
	}

	// 2. защита от двойного reversal
	exists, err := uc.reversalChecker.ExistsReversal(tx, target.ID())
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, entity.ErrOperationAlreadyReversed
	}

	// 3. блокируем session
	session, err := uc.sessionLocker.FindByIDForUpdate(tx, target.SessionID())
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, entity.ErrSessionNotFound
	}

	if session.Status() != entity.StatusActive {
		return nil, entity.ErrSessionNotActive
	}

	// 4. проверка инварианта (важно!)
	if target.Type() == entity.OperationBuyIn {
		if target.Chips() > session.TotalChips() {
			return nil, entity.ErrInvalidCashOut
		}
	}

	// 5. применяем
	if err := uc.applyReversal(session, target); err != nil {
		return nil, err
	}

	// 6. создаём reversal
	op, err := entity.NewReversalOperation(
		uc.idGen.New(),
		cmd.RequestID,
		target.SessionID(),
		target.PlayerID(),
		target.Chips(),
		target.ID(),
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	// 7. сохраняем
	if err := uc.opWriter.Save(tx, op); err != nil {
		return nil, err
	}

	if err := uc.sessionWriter.Save(tx, session); err != nil {
		return nil, err
	}

	event, err := NewOperationReversedOutboxEvent(op)
	if err != nil {
		return nil, err
	}
	if err := uc.outboxWriter.Save(tx, event); err != nil {
		return nil, err
	}
	return op, nil
}

func (uc *ReverseOperationUseCase) applyReversal(
	session *entity.Session,
	target *entity.Operation,
) error {

	switch target.Type() {
	case entity.OperationBuyIn:
		return session.ReverseBuyIn(target.Chips())
	case entity.OperationCashOut:
		return session.ReverseCashOut(target.Chips())
	default:
		return entity.ErrInvalidOperation
	}
}
