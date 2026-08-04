package usecase

import (
	"context"
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type BuyInUseCase struct {
	helper          *Helper
	sessionLocker   SessionLocker
	txManager       TxManager
	idempotencyRepo IdempotencyRepository
	outboxWriter    OutboxWriter
}

func NewBuyInUseCase(
	helper *Helper,
	sessionLocker SessionLocker,
	txManager TxManager,
	idempotencyRepo IdempotencyRepository,
	outboxWriter OutboxWriter,
) *BuyInUseCase {
	return &BuyInUseCase{
		helper:          helper,
		sessionLocker:   sessionLocker,
		txManager:       txManager,
		idempotencyRepo: idempotencyRepo,
		outboxWriter:    outboxWriter,
	}
}

func (uc *BuyInUseCase) Execute(ctx context.Context, cmd command.BuyInCommand) (OperationAcknowledgement, error) {
	var acknowledgement OperationAcknowledgement
	err := uc.txManager.RunInTx(ctx, func(tx Tx) error {
		op, duplicate, err := IdempotentOperation(tx, uc.idempotencyRepo, uc.helper.opReader, cmd.RequestID, OperationFingerprint{
			Type: entity.OperationBuyIn, SessionID: cmd.SessionID, PlayerID: cmd.PlayerID, Chips: cmd.Chips,
		}, func() (*entity.Operation, error) { return uc.execute(tx, cmd) })
		if err != nil {
			return err
		}
		acknowledgement = NewOperationAcknowledgement(op, duplicate)
		return nil
	})
	return acknowledgement, err
}

func (uc *BuyInUseCase) execute(tx Tx, cmd command.BuyInCommand) (*entity.Operation, error) {
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

	exists, err := uc.helper.playerRepo.Exists(tx, cmd.PlayerID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, entity.ErrPlayerNotFound
	}

	// 3. бизнес-операция
	if err := session.BuyIn(cmd.Chips); err != nil {
		return nil, err
	}

	// 4. создаём operation
	op, err := uc.helper.BuildOperation(
		cmd.RequestID,
		cmd.SessionID,
		entity.OperationBuyIn,
		cmd.PlayerID,
		cmd.Chips,
	)
	if err != nil {
		return nil, err
	}

	// 5. сохраняем
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
