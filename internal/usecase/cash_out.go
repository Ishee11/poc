package usecase

import (
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type CashOutUseCase struct {
	helper            *Helper
	sessionLocker     SessionLocker
	playerStateReader OperationPlayerStateReader
	txManager         TxManager
	idempotencyRepo   IdempotencyRepository
}

func NewCashOutUseCase(
	helper *Helper,
	sessionLocker SessionLocker,
	playerStateReader OperationPlayerStateReader,
	txManager TxManager,
	idempotencyRepo IdempotencyRepository,
) *CashOutUseCase {
	return &CashOutUseCase{
		helper:            helper,
		sessionLocker:     sessionLocker,
		playerStateReader: playerStateReader,
		txManager:         txManager,
		idempotencyRepo:   idempotencyRepo,
	}
}

func (uc *CashOutUseCase) Execute(cmd command.CashOutCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {
			return uc.execute(tx, cmd)
		})
	})
}

func (uc *CashOutUseCase) execute(tx Tx, cmd command.CashOutCommand) error {
	// 1. блокируем сессию
	session, err := uc.sessionLocker.FindByIDForUpdate(tx, cmd.SessionID)
	if err != nil {
		return err
	}

	if session.Status() != entity.StatusActive {
		return entity.ErrSessionNotActive
	}

	// 2. валидация
	if cmd.Chips <= 0 {
		return entity.ErrInvalidChips
	}

	if cmd.Chips > session.TotalChips() {
		return entity.ErrInvalidCashOut
	}

	// 3. состояние игрока
	state, err := uc.loadPlayerState(tx, cmd.SessionID, cmd.PlayerID)
	if err != nil {
		return err
	}

	if err := state.ValidateInGame(); err != nil {
		return err
	}

	// 4. применяем к домену
	if err := session.CashOut(cmd.Chips); err != nil {
		return err
	}

	// 5. создаём operation
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

	// 6. сохраняем
	if err := uc.helper.opWriter.Save(tx, op); err != nil {
		return err
	}

	return uc.helper.sessionWriter.Save(tx, session)
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
