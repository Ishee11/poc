package usecase

import (
	"strings"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

type RenameDebugPlayerUseCase struct {
	repo      DebugAdminRepository
	txManager TxManager
}

func NewRenameDebugPlayerUseCase(repo DebugAdminRepository, txManager TxManager) *RenameDebugPlayerUseCase {
	return &RenameDebugPlayerUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *RenameDebugPlayerUseCase) Execute(playerID entity.PlayerID, name string) error {
	name = strings.TrimSpace(name)
	if playerID == "" {
		return entity.ErrInvalidPlayerID
	}
	if name == "" {
		return entity.ErrInvalidPlayerName
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return uc.repo.RenamePlayer(tx, playerID, name)
	})
}

type UpdateDebugSessionConfigUseCase struct {
	repo      DebugAdminRepository
	txManager TxManager
}

func NewUpdateDebugSessionConfigUseCase(repo DebugAdminRepository, txManager TxManager) *UpdateDebugSessionConfigUseCase {
	return &UpdateDebugSessionConfigUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *UpdateDebugSessionConfigUseCase) Execute(sessionID entity.SessionID, chipRate int64, bigBlind int64, currency entity.Currency) error {
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

	return uc.txManager.RunInTx(func(tx Tx) error {
		return uc.repo.UpdateSessionConfig(tx, sessionID, chipRate, bigBlind, currency)
	})
}

type DeleteDebugPlayerUseCase struct {
	repo      DebugAdminRepository
	txManager TxManager
}

func NewDeleteDebugPlayerUseCase(repo DebugAdminRepository, txManager TxManager) *DeleteDebugPlayerUseCase {
	return &DeleteDebugPlayerUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *DeleteDebugPlayerUseCase) Execute(playerID entity.PlayerID) error {
	if playerID == "" {
		return entity.ErrInvalidPlayerID
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return uc.repo.DeletePlayer(tx, playerID)
	})
}

type DeleteDebugSessionUseCase struct {
	repo      DebugAdminRepository
	txManager TxManager
}

type DeleteDebugSessionFinishUseCase struct {
	repo      DebugAdminRepository
	txManager TxManager
}

func NewDeleteDebugSessionFinishUseCase(repo DebugAdminRepository, txManager TxManager) *DeleteDebugSessionFinishUseCase {
	return &DeleteDebugSessionFinishUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *DeleteDebugSessionFinishUseCase) Execute(sessionID entity.SessionID) error {
	if sessionID == "" {
		return entity.ErrSessionNotFound
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return uc.repo.DeleteSessionFinish(tx, sessionID)
	})
}

func NewDeleteDebugSessionUseCase(repo DebugAdminRepository, txManager TxManager) *DeleteDebugSessionUseCase {
	return &DeleteDebugSessionUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *DeleteDebugSessionUseCase) Execute(sessionID entity.SessionID) error {
	if sessionID == "" {
		return entity.ErrSessionNotFound
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return uc.repo.DeleteSession(tx, sessionID)
	})
}
