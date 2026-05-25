package usecase

import (
	"context"
	"strings"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

type AdminRenamePlayerUseCase struct {
	repo      AdminRepository
	txManager TxManager
}

func NewAdminRenamePlayerUseCase(repo AdminRepository, txManager TxManager) *AdminRenamePlayerUseCase {
	return &AdminRenamePlayerUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *AdminRenamePlayerUseCase) Execute(ctx context.Context, playerID entity.PlayerID, name string) error {
	name = strings.TrimSpace(name)
	if playerID == "" {
		return entity.ErrInvalidPlayerID
	}
	if name == "" {
		return entity.ErrInvalidPlayerName
	}

	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		return uc.repo.RenamePlayer(tx, playerID, name)
	})
}

type AdminUpdateSessionConfigUseCase struct {
	repo      AdminRepository
	txManager TxManager
}

func NewAdminUpdateSessionConfigUseCase(repo AdminRepository, txManager TxManager) *AdminUpdateSessionConfigUseCase {
	return &AdminUpdateSessionConfigUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *AdminUpdateSessionConfigUseCase) Execute(ctx context.Context, sessionID entity.SessionID, chipRate int64, bigBlind int64, currency entity.Currency) error {
	if sessionID == "" {
		return entity.ErrSessionNotFound
	}
	if _, err := valueobject.NewChipRate(chipRate); err != nil {
		return err
	}
	if bigBlind <= 0 {
		return valueobject.ErrInvalidChips
	}
	if currency != entity.CurrencyRUB && currency != entity.CurrencyUSD {
		currency = entity.CurrencyRUB
	}

	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		return uc.repo.UpdateSessionConfig(tx, sessionID, chipRate, bigBlind, currency)
	})
}

type AdminDeletePlayerUseCase struct {
	repo      AdminRepository
	txManager TxManager
}

func NewAdminDeletePlayerUseCase(repo AdminRepository, txManager TxManager) *AdminDeletePlayerUseCase {
	return &AdminDeletePlayerUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *AdminDeletePlayerUseCase) Execute(ctx context.Context, playerID entity.PlayerID) error {
	if playerID == "" {
		return entity.ErrInvalidPlayerID
	}

	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		return uc.repo.DeletePlayer(tx, playerID)
	})
}

type AdminDeleteSessionUseCase struct {
	repo      AdminRepository
	txManager TxManager
}

func NewAdminDeleteSessionUseCase(repo AdminRepository, txManager TxManager) *AdminDeleteSessionUseCase {
	return &AdminDeleteSessionUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *AdminDeleteSessionUseCase) Execute(ctx context.Context, sessionID entity.SessionID) error {
	if sessionID == "" {
		return entity.ErrSessionNotFound
	}

	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		return uc.repo.DeleteSession(tx, sessionID)
	})
}

type AdminDeleteSessionFinishUseCase struct {
	repo      AdminRepository
	txManager TxManager
}

func NewAdminDeleteSessionFinishUseCase(repo AdminRepository, txManager TxManager) *AdminDeleteSessionFinishUseCase {
	return &AdminDeleteSessionFinishUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *AdminDeleteSessionFinishUseCase) Execute(ctx context.Context, sessionID entity.SessionID) error {
	if sessionID == "" {
		return entity.ErrSessionNotFound
	}

	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		return uc.repo.DeleteSessionFinish(tx, sessionID)
	})
}
