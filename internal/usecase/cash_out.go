package usecase

import (
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type CashOutUseCase struct {
	helper *Helper

	playerStateReader OperationPlayerStateReader
	projection        ProjectionRepository

	txManager       TxManager
	idempotencyRepo IdempotencyRepository
}

func NewCashOutUseCase(
	helper *Helper,
	playerStateReader OperationPlayerStateReader,
	projection ProjectionRepository,
	txManager TxManager,
	idempotencyRepo IdempotencyRepository,
) *CashOutUseCase {
	return &CashOutUseCase{
		helper:            helper,
		playerStateReader: playerStateReader,
		projection:        projection,
		txManager:         txManager,
		idempotencyRepo:   idempotencyRepo,
	}
}

func (uc *CashOutUseCase) Execute(cmd command.CashOutCommand) error {
	if cmd.Chips <= 0 {
		return entity.ErrInvalidChips
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {
			return uc.execute(tx, cmd)
		})
	})
}

func (uc *CashOutUseCase) execute(tx Tx, cmd command.CashOutCommand) error {
	// 1. session
	session, err := uc.helper.GetActiveSession(tx, cmd.SessionID)
	if err != nil {
		return err
	}

	// 2. player
	playerID, err := uc.helper.GetOrCreatePlayer(
		tx,
		cmd.SessionID,
		cmd.PlayerName,
	)
	if err != nil {
		return err
	}

	// 3. проверка состояния игрока
	if err := uc.validatePlayerState(tx, cmd.SessionID, playerID); err != nil {
		return err
	}

	// 4. проверка фишек на столе
	if err := uc.validateTableChips(tx, cmd.SessionID, cmd.Chips); err != nil {
		return err
	}

	// 5. создаём операцию
	op, err := uc.helper.BuildOperation(
		cmd.RequestID,
		cmd.SessionID,
		entity.OperationCashOut,
		playerID,
		cmd.Chips,
	)
	if err != nil {
		return err
	}

	// 6. применяем к домену
	if err := session.CashOut(cmd.Chips); err != nil {
		return err
	}

	// 7. сохраняем
	return uc.helper.Save(tx, op, session)
}

func (uc *CashOutUseCase) validatePlayerState(
	tx Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) error {

	lastOpType, found, err := uc.playerStateReader.GetLastOperationType(
		tx,
		sessionID,
		playerID,
	)
	if err != nil {
		return err
	}

	if !found {
		return entity.ErrPlayerNotInGame
	}

	if lastOpType != entity.OperationBuyIn {
		return entity.ErrInvalidOperation
	}

	return nil
}

func (uc *CashOutUseCase) validateTableChips(
	tx Tx,
	sessionID entity.SessionID,
	chips int64,
) error {

	aggr, err := uc.projection.GetSessionAggregates(tx, sessionID)
	if err != nil {
		return err
	}

	tableChips := aggr.TotalBuyIn - aggr.TotalCashOut
	if chips > tableChips {
		return entity.ErrInvalidCashOut
	}

	return nil
}
