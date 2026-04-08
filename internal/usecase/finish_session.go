package usecase

import (
	"github.com/ishee11/poc/internal/entity"
)

type FinishSessionCommand struct {
	SessionID entity.SessionID
}

type FinishSessionUseCase struct {
	opRepo      OperationRepository
	sessionRepo SessionRepository
	txManager   TxManager
}

func (uc *FinishSessionUseCase) Execute(cmd FinishSessionCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {
		session, err := uc.sessionRepo.FindByID(tx, cmd.SessionID)
		if err != nil {
			return err
		}

		if session.Status() == entity.StatusFinished {
			return nil
		}

		if session.Status() != entity.StatusActive {
			return entity.ErrSessionNotActive
		}

		sessionAggregates, err := uc.opRepo.GetSessionAggregates(tx, cmd.SessionID)
		if err != nil {
			return err
		}

		tableChips := sessionAggregates.TotalBuyIn - sessionAggregates.TotalCashOut
		if tableChips != 0 {
			return entity.ErrTableNotSettled
		}

		if err := session.Finish(); err != nil {
			return err
		}

		if err := uc.sessionRepo.Save(tx, session); err != nil {
			return err
		}

		return uc.sessionRepo.Save(tx, session)
	})
}
