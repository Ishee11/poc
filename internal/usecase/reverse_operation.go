package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type ReverseOperationUseCase struct {
	opWriter OperationWriter

	opReader        OperationReader
	reversalChecker OperationReversalChecker

	sessionReader SessionReader
	sessionWriter SessionWriter

	txManager       TxManager
	idGen           OperationIDGenerator
	idempotencyRepo IdempotencyRepository
}

func NewReverseOperationUseCase(
	opWriter OperationWriter,
	opReader OperationReader,
	reversalChecker OperationReversalChecker,
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	txManager TxManager,
	idGen OperationIDGenerator,
	idempotencyRepo IdempotencyRepository,
) *ReverseOperationUseCase {
	return &ReverseOperationUseCase{
		opWriter:        opWriter,
		opReader:        opReader,
		reversalChecker: reversalChecker,
		sessionReader:   sessionReader,
		sessionWriter:   sessionWriter,
		txManager:       txManager,
		idGen:           idGen,
		idempotencyRepo: idempotencyRepo,
	}
}

func (uc *ReverseOperationUseCase) Execute(cmd command.ReverseOperationCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {
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

	// 2. нельзя отменять reversal
	if target.Type() == entity.OperationReversal {
		return entity.ErrInvalidOperation
	}

	// 3. защита от двойного reversal
	exists, err := uc.reversalChecker.ExistsReversal(tx, target.ID())
	if err != nil {
		return err
	}
	if exists {
		return entity.ErrOperationAlreadyReversed
	}

	// 4. session
	session, err := uc.sessionReader.FindByID(tx, target.SessionID())
	if err != nil {
		return err
	}

	if session.Status() != entity.StatusActive {
		return entity.ErrSessionNotActive
	}

	// 5. создаём reversal
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

	// 6. сохраняем операцию
	if err := uc.opWriter.Save(tx, op); err != nil {
		return err
	}

	// 7. инверсия домена
	if err := uc.applyReversal(session, target); err != nil {
		return err
	}

	// 8. сохраняем session
	return uc.sessionWriter.Save(tx, session)
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
