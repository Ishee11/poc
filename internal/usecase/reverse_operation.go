package usecase

import (
	"errors"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type ReverseOperationCommand struct {
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
		// 1. найти target operation
		target, err := uc.opRepo.GetByID(tx, cmd.TargetOperationID)
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
		exists, err := uc.opRepo.ExistsReversal(tx, target.ID())
		if err != nil {
			return err
		}
		if exists {
			return entity.ErrOperationAlreadyReversed
		}

		// 4. загрузить session
		session, err := uc.sessionRepo.FindByID(tx, target.SessionID())
		if err != nil {
			return err
		}

		if session.Status() != entity.StatusActive {
			return entity.ErrSessionNotActive
		}

		// 5. создать reversal
		now := time.Now()

		op, err := entity.NewReversalOperation(
			cmd.OperationID,
			target.SessionID(),
			target.PlayerID(),
			target.Chips(),
			target.ID(),
			now,
		)
		if err != nil {
			return err
		}

		// 6. сохранить операцию
		err = uc.opRepo.Save(tx, op)
		if err != nil {
			if errors.Is(err, entity.ErrDuplicateOperation) {
				return nil
			}
			return err
		}

		// 7. обновить session (инверсия)
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

		// 8. сохранить session
		if err := uc.sessionRepo.Save(tx, session); err != nil {
			return err
		}

		return nil
	})
}
