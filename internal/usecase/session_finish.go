package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type FinishSessionUseCase struct {
	projection      ProjectionRepository
	sessionReader   SessionReader
	sessionWriter   SessionWriter
	txManager       TxManager
	idempotencyRepo IdempotencyRepository
}

func NewFinishSessionUseCase(
	projection ProjectionRepository,
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	txManager TxManager,
	idempotencyRepo IdempotencyRepository,
) *FinishSessionUseCase {
	return &FinishSessionUseCase{
		projection:      projection,
		sessionReader:   sessionReader,
		sessionWriter:   sessionWriter,
		txManager:       txManager,
		idempotencyRepo: idempotencyRepo,
	}
}

func (uc *FinishSessionUseCase) Execute(cmd command.FinishSessionCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {
			return uc.execute(tx, cmd)
		})
	})
}

func (uc *FinishSessionUseCase) execute(tx Tx, cmd command.FinishSessionCommand) error {
	session, err := uc.sessionReader.FindByID(tx, cmd.SessionID)
	if err != nil {
		return err
	}

	if session.Status() == entity.StatusFinished {
		return entity.ErrSessionFinished
	}

	if session.Status() != entity.StatusActive {
		return entity.ErrSessionNotActive
	}

	aggr, err := uc.projection.GetSessionAggregates(tx, cmd.SessionID)
	if err != nil {
		return err
	}

	if aggr.TotalBuyIn != aggr.TotalCashOut {
		return &entity.SessionNotBalancedError{
			RemainingChips: aggr.TotalBuyIn - aggr.TotalCashOut,
		}
	}

	if err := session.Finish(time.Now()); err != nil {
		return err
	}

	return uc.sessionWriter.Save(tx, session)
}
