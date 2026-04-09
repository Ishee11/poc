package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type IDGenerator interface {
	New() entity.OperationID
}

type CashOutUseCase struct {
	opRepo      OperationRepository
	sessionRepo SessionRepository
	txManager   TxManager
	idGen       IDGenerator
}

type CashOutCommand struct {
	RequestID string
	SessionID entity.SessionID
	PlayerID  entity.PlayerID
	Chips     int64
}

func (uc *CashOutUseCase) Execute(cmd CashOutCommand) error {
	if cmd.Chips <= 0 {
		return entity.ErrInvalidChips
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, cmd.RequestID, func() error {

			// 2. получаем сессию
			session, err := uc.sessionRepo.FindByID(tx, cmd.SessionID)
			if err != nil {
				return err
			}

			if session.Status() != entity.StatusActive {
				return entity.ErrSessionNotActive
			}

			// 3. проверяем состояние игрока
			lastOpType, found, err := uc.opRepo.GetLastOperationType(tx, cmd.SessionID, cmd.PlayerID)
			if err != nil {
				return err
			}

			if !found {
				return entity.ErrPlayerNotInGame
			}

			if lastOpType != entity.OperationBuyIn {
				return entity.ErrInvalidOperation
			}

			// 4. агрегаты по сессии
			aggr, err := uc.opRepo.GetSessionAggregates(tx, cmd.SessionID)
			if err != nil {
				return err
			}

			tableChips := aggr.TotalBuyIn - aggr.TotalCashOut
			if cmd.Chips > tableChips {
				return entity.ErrInvalidCashOut
			}

			// 5. создаём операцию
			opID := uc.idGen.New()

			op, err := entity.NewOperation(
				opID,
				cmd.RequestID,
				cmd.SessionID,
				entity.OperationCashOut,
				cmd.PlayerID,
				cmd.Chips,
				time.Now(),
			)
			if err != nil {
				return err
			}

			// 6. сохраняем
			if err := uc.opRepo.Save(tx, op); err != nil {
				return err
			}

			// 7. обновляем сессию
			if err := session.CashOut(cmd.Chips); err != nil {
				return err
			}

			return uc.sessionRepo.Save(tx, session)
		})
	})
}
