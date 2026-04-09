package usecase

import (
	"errors"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type CashOutCommand struct {
	OperationID entity.OperationID
	SessionID   entity.SessionID
	PlayerID    entity.PlayerID
	Chips       int64
}

type CashOutUseCase struct {
	opRepo      OperationRepository
	sessionRepo SessionRepository
	txManager   TxManager
}

func (uc *CashOutUseCase) Execute(cmd CashOutCommand) error {
	if cmd.Chips <= 0 {
		return entity.ErrInvalidChips
	}
	return uc.txManager.RunInTx(func(tx Tx) error {
		session, err := uc.sessionRepo.FindByID(tx, cmd.SessionID)
		if err != nil {
			return err
		}

		if session.Status() != entity.StatusActive {
			return entity.ErrSessionNotActive
		}

		opType, found, err := uc.opRepo.GetLastOperationType(tx, cmd.SessionID, cmd.PlayerID)
		if err != nil {
			return err
		}

		if !found {
			return entity.ErrPlayerNotInGame
		}

		if opType != entity.OperationBuyIn {
			return entity.ErrInvalidOperation
		}

		sessionAggregates, err := uc.opRepo.GetSessionAggregates(tx, cmd.SessionID)
		if err != nil {
			return err
		}

		tableChips := sessionAggregates.TotalBuyIn - sessionAggregates.TotalCashOut
		if cmd.Chips > tableChips {
			return entity.ErrInvalidCashOut
		}

		date := time.Now()
		op, err := entity.NewOperation(
			cmd.OperationID,
			cmd.SessionID,
			entity.OperationCashOut,
			cmd.PlayerID,
			cmd.Chips,
			date)
		if err != nil {
			return err
		}

		err = uc.opRepo.Save(tx, op)
		if err != nil {
			if errors.Is(err, entity.ErrDuplicateOperation) {
				return nil
			}
			return err
		}

		if err := session.CashOut(cmd.Chips); err != nil {
			return err
		}

		if err := uc.sessionRepo.Save(tx, session); err != nil {
			return err
		}

		return nil
	})
}
