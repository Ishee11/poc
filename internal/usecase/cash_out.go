package usecase

import (
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type CashOutUseCase struct {
	helper            *Helper
	playerStateReader OperationPlayerStateReader
	projection        ProjectionRepository
	txManager         TxManager
	idempotencyRepo   IdempotencyRepository
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

	// 3. проверка состояния игрока
	state, err := uc.loadPlayerState(tx, cmd.SessionID, cmd.PlayerID)
	if err != nil {
		return err
	}

	if err := state.ValidateInGame(); err != nil {
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
		cmd.PlayerID,
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

func (uc *CashOutUseCase) loadPlayerState(
	tx Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) (entity.PlayerState, error) {

	lastOpType, found, err := uc.playerStateReader.GetLastOperationType(
		tx,
		sessionID,
		playerID,
	)
	if err != nil {
		return entity.PlayerState{}, err
	}

	return entity.NewPlayerState(
		playerID,
		lastOpType,
		found,
	), nil
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
