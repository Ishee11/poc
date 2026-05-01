package usecase

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
)

type LinkUserPlayerCommand struct {
	UserID   entity.AuthUserID
	PlayerID entity.PlayerID
}

type ListUnlinkedPlayersQuery struct {
	Limit  int
	Offset int
}

type UserPlayerLinksUseCase struct {
	linkRepo   UserPlayerLinkRepository
	playerRepo PlayerRepository
	txManager  TxManager
}

func NewUserPlayerLinksUseCase(
	linkRepo UserPlayerLinkRepository,
	playerRepo PlayerRepository,
	txManager TxManager,
) *UserPlayerLinksUseCase {
	return &UserPlayerLinksUseCase{
		linkRepo:   linkRepo,
		playerRepo: playerRepo,
		txManager:  txManager,
	}
}

func (uc *UserPlayerLinksUseCase) LinkPlayer(ctx context.Context, cmd LinkUserPlayerCommand) error {
	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		exists, err := uc.playerRepo.Exists(tx, cmd.PlayerID)
		if err != nil {
			return err
		}
		if !exists {
			return entity.ErrPlayerNotFound
		}

		linked, err := uc.linkRepo.IsPlayerLinked(tx, cmd.PlayerID)
		if err != nil {
			return err
		}
		if linked {
			linkedToUser, err := uc.linkRepo.IsPlayerLinkedToUser(tx, cmd.UserID, cmd.PlayerID)
			if err != nil {
				return err
			}
			if linkedToUser {
				return nil
			}
			return entity.ErrPlayerAlreadyLinked
		}

		return uc.linkRepo.LinkPlayer(tx, cmd.UserID, cmd.PlayerID)
	})
}

func (uc *UserPlayerLinksUseCase) UnlinkPlayer(ctx context.Context, cmd LinkUserPlayerCommand) error {
	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		linkedToUser, err := uc.linkRepo.IsPlayerLinkedToUser(tx, cmd.UserID, cmd.PlayerID)
		if err != nil {
			return err
		}
		if !linkedToUser {
			return entity.ErrUserPlayerNotLinked
		}

		return uc.linkRepo.UnlinkPlayer(tx, cmd.UserID, cmd.PlayerID)
	})
}

func (uc *UserPlayerLinksUseCase) ListUserPlayers(ctx context.Context, userID entity.AuthUserID) ([]PlayerDTO, error) {
	var result []PlayerDTO

	err := uc.txManager.RunInTx(ctx, func(tx Tx) error {
		var err error
		result, err = uc.linkRepo.ListUserPlayers(tx, userID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *UserPlayerLinksUseCase) ListUnlinkedPlayers(ctx context.Context, q ListUnlinkedPlayersQuery) ([]PlayerDTO, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	var result []PlayerDTO
	err := uc.txManager.RunInTx(ctx, func(tx Tx) error {
		var err error
		result, err = uc.linkRepo.ListUnlinkedPlayers(tx, limit, offset)
		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
