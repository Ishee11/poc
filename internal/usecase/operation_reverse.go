// NOTE:
// - No DB-level protection for reversal uniqueness (acceptable for single-user scenario)
// - Idempotency is best-effort (may skip execution after partial failure)
package usecase

import (
	"context"
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

func (uc *ReverseOperationUseCase) Execute(ctx context.Context, cmd command.ReverseOperationCommand) error {
	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {
			return uc.execute(tx, cmd)
		})
	})
}

func (uc *ReverseOperationUseCase) execute(tx Tx, cmd command.ReverseOperationCommand) error {
	// 1. target operation
	target, err := uc.opReader.GetByID(tx, cmd.TargetOperationID)
	if err != nil {
		return err
	}
	if target == nil {
		return entity.ErrOperationNotFound
	}

	if target.Type() == entity.OperationReversal {
		return entity.ErrInvalidOperation
	}

	// 2. защита от двойного reversal
	exists, err := uc.reversalChecker.ExistsReversal(tx, target.ID())
	if err != nil {
		return err
	}
	if exists {
		return entity.ErrOperationAlreadyReversed
	}

	// 3. блокируем session
	session, err := uc.sessionLocker.FindByIDForUpdate(tx, target.SessionID())
	if err != nil {
		return err
	}
	if session == nil {
		return entity.ErrSessionNotFound
	}

	if session.Status() != entity.StatusActive {
		return entity.ErrSessionNotActive
	}

	// 4. проверка инварианта (важно!)
	if target.Type() == entity.OperationBuyIn {
		if target.Chips() > session.TotalChips() {
			return entity.ErrInvalidCashOut
		}
	}

	// 5. применяем
	if err := uc.applyReversal(session, target); err != nil {
		return err
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
		return err
	}

	// 7. сохраняем
	if err := uc.opWriter.Save(tx, op); err != nil {
		return err
	}

	if err := uc.sessionWriter.Save(tx, session); err != nil {
		return err
	}

	event, err := NewOperationReversedOutboxEvent(op)
	if err != nil {
		return err
	}

	return uc.outboxWriter.Save(tx, event)
}

func (uc *ReverseOperationUseCase) applyReversal(
	session *entity.Session,
	target *entity.Operation,
) error {

	switch target.Type() {
	case entity.OperationBuyIn:
		return session.CashOut(target.Chips())
	case entity.OperationCashOut:
		return session.BuyIn(target.Chips())
	default:
		return entity.ErrInvalidOperation
	}
}
