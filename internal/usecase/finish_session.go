package usecase

import (
	"github.com/ishee11/poc/internal/entity"
)

type SessionNotBalancedError struct {
	RemainingChips int64
}

func (e *SessionNotBalancedError) Error() string {
	return "session not balanced"
}

type FinishSessionCommand struct {
	SessionID entity.SessionID
}

type FinishSessionUseCase struct {
	projection    ProjectionRepository
	sessionReader SessionReader
	sessionWriter SessionWriter
	txManager     TxManager
}

func NewFinishSessionUseCase(
	projection ProjectionRepository,
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	txManager TxManager,
) *FinishSessionUseCase {
	return &FinishSessionUseCase{
		projection:    projection,
		sessionReader: sessionReader,
		sessionWriter: sessionWriter,
		txManager:     txManager,
	}
}

func (uc *FinishSessionUseCase) Execute(cmd FinishSessionCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {

		// 1. получаем session
		session, err := uc.sessionReader.FindByID(tx, cmd.SessionID)
		if err != nil {
			return err
		}

		if session.Status() == entity.StatusFinished {
			return nil // idempotent
		}

		if session.Status() != entity.StatusActive {
			return entity.ErrSessionNotActive
		}

		// 2. проверяем баланс через projection
		aggr, err := uc.projection.GetSessionAggregates(tx, cmd.SessionID)
		if err != nil {
			return err
		}

		if aggr.TotalBuyIn != aggr.TotalCashOut {
			return &SessionNotBalancedError{
				RemainingChips: aggr.TotalBuyIn - aggr.TotalCashOut,
			}
		}

		// 3. завершаем
		if err := session.Finish(); err != nil {
			return err
		}

		return uc.sessionWriter.Save(tx, session)
	})
}
