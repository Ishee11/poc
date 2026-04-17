package usecase

import (
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type BuyInUseCase struct {
	helper          *Helper
	txManager       TxManager
	idempotencyRepo IdempotencyRepository
}

func NewBuyInUseCase(
	helper *Helper,
	txManager TxManager,
	idempotencyRepo IdempotencyRepository,
) *BuyInUseCase {
	return &BuyInUseCase{
		helper:          helper,
		txManager:       txManager,
		idempotencyRepo: idempotencyRepo,
	}
}

func (uc *BuyInUseCase) Execute(cmd command.BuyInCommand) error {
	if cmd.Chips <= 0 {
		return entity.ErrInvalidChips
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {
			return uc.execute(tx, cmd)
		})
	})
}

func (uc *BuyInUseCase) execute(tx Tx, cmd command.BuyInCommand) error {
	session, err := uc.helper.GetActiveSession(tx, cmd.SessionID)
	if err != nil {
		return err
	}

	if err := session.BuyIn(cmd.Chips); err != nil {
		return err
	}

	op, err := uc.helper.BuildOperation(cmd.RequestID, cmd.SessionID, entity.OperationBuyIn, cmd.PlayerID, cmd.Chips)
	if err != nil {
		return err
	}

	return uc.helper.Save(tx, op, session)
}
