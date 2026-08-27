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

func (s *SessionAccessService) RequireMutation(ctx context.Context, query SessionAccessQuery) error {
	if query.SessionID == "" {
		return entity.ErrSessionNotFound
	}
	if query.ViewerUserID == nil {
		return entity.ErrUnauthorized
	}

	allowed, err := s.CanMutate(ctx, query)
	if err != nil {
		return err
	}
	if !allowed {
		return entity.ErrForbidden
	}
	return nil
}

func (s *SessionAccessService) CanMutate(ctx context.Context, query SessionAccessQuery) (bool, error) {
	if query.SessionID == "" || query.ViewerUserID == nil {
		return false, nil
	}
	allowed := false
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		var err error
		allowed, err = s.repo.CanMutateSession(tx, query.SessionID, *query.ViewerUserID, query.ViewerIsAdmin)
		return err
	})
	return allowed, err
}
