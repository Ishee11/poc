package usecase

import (
	"context"
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type CashOutUseCase struct {
	helper            *Helper
	sessionLocker     SessionLocker
	playerStateReader OperationPlayerStateReader
	txManager         TxManager
	idempotencyRepo   IdempotencyRepository
	outboxWriter      OutboxWriter
}

func NewCashOutUseCase(
	helper *Helper,
	sessionLocker SessionLocker,
	playerStateReader OperationPlayerStateReader,
	txManager TxManager,
	idempotencyRepo IdempotencyRepository,
	outboxWriter OutboxWriter,
) *CashOutUseCase {
	return &CashOutUseCase{
		helper:            helper,
		sessionLocker:     sessionLocker,
		playerStateReader: playerStateReader,
		txManager:         txManager,
		idempotencyRepo:   idempotencyRepo,
		outboxWriter:      outboxWriter,
	}
}

func (uc *CashOutUseCase) Execute(ctx context.Context, cmd command.CashOutCommand) (OperationAcknowledgement, error) {
	var acknowledgement OperationAcknowledgement
	err := uc.txManager.RunInTx(ctx, func(tx Tx) error {
		op, duplicate, err := IdempotentOperation(tx, uc.idempotencyRepo, uc.helper.opReader, cmd.RequestID, OperationFingerprint{
			Type: entity.OperationCashOut, SessionID: cmd.SessionID, PlayerID: cmd.PlayerID, Chips: cmd.Chips,
		}, func() (*entity.Operation, error) { return uc.execute(tx, cmd) })
		if err != nil {
			return err
		}
		acknowledgement = NewOperationAcknowledgement(op, duplicate)
		return nil
	})
	return acknowledgement, err
}

func (uc *CashOutUseCase) execute(tx Tx, cmd command.CashOutCommand) (*entity.Operation, error) {
	// 1. блокируем сессию
	session, err := uc.sessionLocker.FindByIDForUpdate(tx, cmd.SessionID)
	if err != nil {
		return nil, err
	}

	if session.Status() != entity.StatusActive {
		return nil, entity.ErrSessionNotActive
	}

	// 2. валидация
	if cmd.Chips <= 0 {
		return nil, entity.ErrInvalidChips
	}

	if cmd.Chips > session.TotalChips() {
		return nil, entity.ErrInvalidCashOut
	}

	// 3. состояние игрока
	state, err := uc.loadPlayerState(tx, cmd.SessionID, cmd.PlayerID)
	if err != nil {
		return nil, err
	}

	if err := state.ValidateInGame(); err != nil {
		return nil, err
	}

	// 4. применяем к домену
	if err := session.CashOut(cmd.Chips); err != nil {
		return nil, err
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
		return nil, err
	}

	// 6. сохраняем
	if err := uc.helper.opWriter.Save(tx, op); err != nil {
		return nil, err
	}

	if err := uc.helper.sessionWriter.Save(tx, session); err != nil {
		return nil, err
	}

	event, err := NewOperationCreatedOutboxEvent(op)
	if err != nil {
		return nil, err
	}
	if err := uc.outboxWriter.Save(tx, event); err != nil {
		return nil, err
	}
	return op, nil
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
