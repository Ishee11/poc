package usecase

import (
	"context"
	"strings"

	"github.com/ishee11/poc/internal/entity"
)

type AdminListAccountsQuery struct {
	Query  string
	Limit  int
	Offset int
}

type AdminAccountOwnershipService struct {
	repo      AdminAccountOwnershipRepository
	txManager TxManager
}

func NewAdminAccountOwnershipService(
	repo AdminAccountOwnershipRepository,
	txManager TxManager,
) *AdminAccountOwnershipService {
	return &AdminAccountOwnershipService{repo: repo, txManager: txManager}
}

func (s *AdminAccountOwnershipService) ListAccounts(
	ctx context.Context,
	query AdminListAccountsQuery,
) (AdminAccountList, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	result := AdminAccountList{Limit: limit, Offset: offset, Accounts: make([]AccountOwnershipDTO, 0)}
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		accounts, total, err := s.repo.ListAccounts(tx, strings.TrimSpace(query.Query), limit, offset)
		if err != nil {
			return err
		}
		result.Accounts = accounts
		result.Total = total
		return nil
	})
	return result, err
}

func (s *AdminAccountOwnershipService) Replace(
	ctx context.Context,
	userID entity.AuthUserID,
	playerID entity.PlayerID,
) (OwnershipChange, error) {
	change := OwnershipChange{TargetUserID: userID}
	if userID == "" {
		return change, entity.ErrAuthUserNotFound
	}
	if playerID == "" {
		return change, entity.ErrInvalidPlayerID
	}

	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		if err := s.repo.LockUser(tx, userID); err != nil {
			return err
		}
		if err := s.repo.LockPlayer(tx, playerID); err != nil {
			return err
		}

		current, err := s.repo.FindUserPlayer(tx, userID)
		if err != nil {
			return err
		}
		if current != nil {
			oldID := current.ID
			change.OldPlayerID = &oldID
			if current.ID == playerID {
				change.NewPlayerID = &oldID
				return nil
			}
		}

		owner, err := s.repo.FindPlayerOwner(tx, playerID)
		if err != nil {
			return err
		}
		if owner != nil && *owner != userID {
			return entity.ErrPlayerAlreadyLinked
		}
		if current != nil {
			if err := s.repo.UnlinkPlayer(tx, userID, current.ID); err != nil {
				return err
			}
		}
		if err := s.repo.LinkPlayer(tx, userID, playerID); err != nil {
			return err
		}
		newID := playerID
		change.NewPlayerID = &newID
		return nil
	})
	return change, err
}

func (s *AdminAccountOwnershipService) Clear(
	ctx context.Context,
	userID entity.AuthUserID,
) (OwnershipChange, error) {
	change := OwnershipChange{TargetUserID: userID}
	if userID == "" {
		return change, entity.ErrAuthUserNotFound
	}

	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		if err := s.repo.LockUser(tx, userID); err != nil {
			return err
		}
		current, err := s.repo.FindUserPlayer(tx, userID)
		if err != nil {
			return err
		}
		if current == nil {
			return nil
		}
		oldID := current.ID
		change.OldPlayerID = &oldID
		return s.repo.UnlinkPlayer(tx, userID, current.ID)
	})
	return change, err
}
