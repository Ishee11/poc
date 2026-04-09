package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type ReverseOperationCommand struct {
	RequestID         string
	OperationID       entity.OperationID
	TargetOperationID entity.OperationID
}

type ReverseOperationUseCase struct {
	opRepo      OperationRepository
	sessionRepo SessionRepository
	txManager   TxManager
}

func (uc *ReverseOperationUseCase) Execute(cmd ReverseOperationCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {

		// 1. идемпотентность (главное правило)
		existing, err := uc.opRepo.GetByRequestID(tx, cmd.RequestID)
		if err != nil {
			return err
		}
		if existing != nil {
			return nil
		}

		// 2. найти target operation
		target, err := uc.opRepo.GetByID(tx, cmd.TargetOperationID)
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
		exists, err := uc.opRepo.ExistsReversal(tx, target.ID())
		if err != nil {
			return err
		}
		if exists {
			return entity.ErrOperationAlreadyReversed
		}

		// 5. загрузить session
		session, err := uc.sessionRepo.FindByID(tx, target.SessionID())
		if err != nil {
			return err
		}

		if session.Status() != entity.StatusActive {
			return entity.ErrSessionNotActive
		}

		// 6. создать reversal (FIX: правильный порядок аргументов)
		op, err := entity.NewReversalOperation(
			cmd.OperationID,
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
		if err := uc.opRepo.Save(tx, op); err != nil {
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
		if err := uc.sessionRepo.Save(tx, session); err != nil {
			return err
		}

		return nil
	})
}
