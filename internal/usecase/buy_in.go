package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type BuyInUseCase struct {
	opRepo      OperationRepository
	sessionRepo SessionRepository
	txManager   TxManager
}

func (uc *BuyInUseCase) Execute(cmd BuyInCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {
		date := time.Now()

		op, err := entity.NewOperation(
			cmd.OperationID,
			cmd.SessionID,
			entity.OperationBuyIn,
			cmd.PlayerID,
			cmd.Chips,
			date)
		if err != nil {
			return err
		}

		err = uc.opRepo.Save(tx, op)
		if err != nil {
			return err
		}

		session, err := uc.sessionRepo.FindByID(tx, cmd.SessionID)
		if err != nil {
			return err
		}

		if err := session.BuyIn(cmd.Chips); err != nil {
			return err
		}

		if err := uc.sessionRepo.Save(tx, session); err != nil {
			return err
		}

		return nil
	})
}
