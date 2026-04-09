package usecase

import (
	"github.com/ishee11/poc/internal/entity"
)

type FinishSessionCommand struct {
	SessionID entity.SessionID
}

type FinishSessionUseCase struct {
	aggregateReader OperationAggregateReader

	sessionReader SessionReader
	sessionWriter SessionWriter

	txManager TxManager
}

func (uc *FinishSessionUseCase) Execute(cmd FinishSessionCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {
		session, err := uc.sessionReader.FindByID(tx, cmd.SessionID)
		if err != nil {
			return err
		}

		// идемпотентность через state
		if session.Status() == entity.StatusFinished {
			return nil
		}

		if session.Status() != entity.StatusActive {
			return entity.ErrSessionNotActive
		}

		sessionAggregates, err := uc.aggregateReader.GetSessionAggregates(tx, cmd.SessionID)
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

		return uc.sessionWriter.Save(tx, session)
	})
}
