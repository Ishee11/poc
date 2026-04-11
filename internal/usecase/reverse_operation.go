package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type ReverseOperationCommand struct {
	RequestID         string
	TargetOperationID entity.OperationID
}

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

func (uc *ReverseOperationUseCase) Execute(cmd ReverseOperationCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {

			// 2. найти target operation
			target, err := uc.opReader.GetByID(tx, cmd.TargetOperationID)
			if err != nil {
				return err
			}
			if target == nil {
				return entity.ErrOperationNotFound
			}

			// 3. нельзя отменять reversal
			if target.Type() == entity.OperationReversal {
				return entity.ErrInvalidOperation
			}

			// 4. защита от двойного reversal (доменная)
			exists, err := uc.reversalChecker.ExistsReversal(tx, target.ID())
			if err != nil {
				return err
			}
			if exists {
				return entity.ErrOperationAlreadyReversed
			}

			// 5. загрузить session
			session, err := uc.sessionReader.FindByID(tx, target.SessionID())
			if err != nil {
				return err
			}

			if session.Status() != entity.StatusActive {
				return entity.ErrSessionNotActive
			}

			// 6. создать reversal (FIX: правильный порядок аргументов)
			opID := uc.idGen.New()
			op, err := entity.NewReversalOperation(
				opID,
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

			// 7. сохранить операцию
			if err := uc.opWriter.Save(tx, op); err != nil {
				return err
			}

			// 8. применить инверсию
			switch target.Type() {
			case entity.OperationBuyIn:
				if err := session.CashOut(target.Chips()); err != nil {
					return err
				}
			case entity.OperationCashOut:
				if err := session.BuyIn(target.Chips()); err != nil {
					return err
				}
			}

			// 9. сохранить session
			if err := uc.sessionWriter.Save(tx, session); err != nil {
				return err
			}

			return nil
		})
	})
}
