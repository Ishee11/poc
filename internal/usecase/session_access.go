package usecase

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
)

type SessionAccessService struct {
	repo      SessionAccessRepository
	txManager TxManager
}

func NewSessionAccessService(repo SessionAccessRepository, txManager TxManager) *SessionAccessService {
	return &SessionAccessService{repo: repo, txManager: txManager}
}

func (s *SessionAccessService) RequireView(ctx context.Context, query SessionAccessQuery) error {
	if query.SessionID == "" {
		return entity.ErrSessionNotFound
	}

	allowed := false
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		var err error
		allowed, err = s.repo.CanViewSession(tx, query.SessionID, SessionAccessFilter{
			ViewerUserID:  query.ViewerUserID,
			ViewerIsAdmin: query.ViewerIsAdmin,
			GuestPlayerID: query.GuestPlayerID,
		})
		return err
	})
	if err != nil {
		return err
	}
	if !allowed {
		return entity.ErrForbidden
	}
	return nil
}
