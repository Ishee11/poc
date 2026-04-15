package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type CashOutUseCase struct {
	opWriter OperationWriter

	playerStateReader OperationPlayerStateReader
	projection        ProjectionRepository

	sessionReader SessionReader
	sessionWriter SessionWriter

	txManager       TxManager
	idGen           OperationIDGenerator
	idempotencyRepo IdempotencyRepository

	playerRepo PlayerRepository
}

func NewCashOutUseCase(
	opWriter OperationWriter,
	playerStateReader OperationPlayerStateReader,
	projection ProjectionRepository,
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	txManager TxManager,
	idGen OperationIDGenerator,
	idempotencyRepo IdempotencyRepository,
	playerRepo PlayerRepository,
) *CashOutUseCase {
	return &CashOutUseCase{
		opWriter:          opWriter,
		playerStateReader: playerStateReader,
		projection:        projection,
		sessionReader:     sessionReader,
		sessionWriter:     sessionWriter,
		txManager:         txManager,
		idGen:             idGen,
		idempotencyRepo:   idempotencyRepo,
		playerRepo:        playerRepo,
	}
}

func (uc *CashOutUseCase) Execute(cmd command.CashOutCommand) error {
	if cmd.Chips <= 0 {
		return entity.ErrInvalidChips
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {

			// 1. session
			session, err := uc.sessionReader.FindByID(tx, cmd.SessionID)
			if err != nil {
				return err
			}

			if session.Status() != entity.StatusActive {
				return entity.ErrSessionNotActive
			}

			// 2. состояние игрока
			playerID, err := uc.playerRepo.GetOrCreate(
				tx,
				cmd.SessionID,
				string(cmd.PlayerID), // name из UI
			)
			if err != nil {
				return err
			}
			lastOpType, found, err := uc.playerStateReader.GetLastOperationType(tx, cmd.SessionID, playerID)
			if err != nil {
				return err
			}

			if !found {
				return entity.ErrPlayerNotInGame
			}

			if lastOpType != entity.OperationBuyIn {
				return entity.ErrInvalidOperation
			}

			// 3. агрегаты (через projection)
			aggr, err := uc.projection.GetSessionAggregates(tx, cmd.SessionID)
			if err != nil {
				return err
			}

			tableChips := aggr.TotalBuyIn - aggr.TotalCashOut
			if cmd.Chips > tableChips {
				return entity.ErrInvalidCashOut
			}

			// 4. создаём операцию
			opID := uc.idGen.New()

			op, err := entity.NewOperation(
				opID,
				cmd.RequestID,
				cmd.SessionID,
				entity.OperationCashOut,
				playerID,
				cmd.Chips,
				time.Now(),
			)
			if err != nil {
				return err
			}

			// 5. сохраняем
			if err := uc.opWriter.Save(tx, op); err != nil {
				return err
			}

			// 6. обновляем session cache
			if err := session.CashOut(cmd.Chips); err != nil {
				return err
			}

			return uc.sessionWriter.Save(tx, session)
		})
	})
}
