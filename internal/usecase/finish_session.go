package usecase

import (
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

func (e *SessionNotBalancedError) Is(target error) bool {
	return target == entity.ErrSessionNotBalanced
}

type SessionNotBalancedError struct {
	RemainingChips int64
}

func (e *SessionNotBalancedError) Error() string {
	return "session not balanced"
}

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
				return &SessionNotBalancedError{
					RemainingChips: aggr.TotalBuyIn - aggr.TotalCashOut,
				}
			}

			if err := session.Finish(); err != nil {
				return err
			}

			return uc.sessionWriter.Save(tx, session)
		})
	})
}
