package usecase

import "github.com/ishee11/poc/internal/entity"

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
