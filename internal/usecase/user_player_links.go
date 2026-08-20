package usecase

import (
	"context"
	"strings"

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
	linkRepo    UserPlayerLinkRepository
	playerRepo  PlayerRepository
	playerIDGen PlayerIDGenerator
	txManager   TxManager
}

func NewUserPlayerLinksUseCase(
	linkRepo UserPlayerLinkRepository,
	playerRepo PlayerRepository,
	playerIDGen PlayerIDGenerator,
	txManager TxManager,
) *UserPlayerLinksUseCase {
	return &UserPlayerLinksUseCase{
		linkRepo:    linkRepo,
		playerRepo:  playerRepo,
		playerIDGen: playerIDGen,
		txManager:   txManager,
	}
}

func (uc *UserPlayerLinksUseCase) LinkPlayer(ctx context.Context, cmd LinkUserPlayerCommand) error {
	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		owned, err := uc.linkRepo.FindUserPlayer(tx, cmd.UserID)
		if err != nil {
			return err
		}
		if owned != nil {
			return entity.ErrAccountAlreadyLinked
		}

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

func (uc *UserPlayerLinksUseCase) ChooseOrCreatePlayer(
	ctx context.Context,
	userID entity.AuthUserID,
	selection PlayerSelection,
) (*PlayerDTO, error) {
	if err := selection.Validate(); err != nil {
		return nil, err
	}

	var result *PlayerDTO
	err := uc.txManager.RunInTx(ctx, func(tx Tx) error {
		player, err := chooseOrCreatePlayerInTx(
			tx,
			userID,
			selection,
			uc.linkRepo,
			uc.playerRepo,
			uc.playerIDGen,
		)
		if err != nil {
			return err
		}
		result = player
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func chooseOrCreatePlayerInTx(
	tx Tx,
	userID entity.AuthUserID,
	selection PlayerSelection,
	linkRepo UserPlayerLinkRepository,
	playerRepo PlayerRepository,
	playerIDGen PlayerIDGenerator,
) (*PlayerDTO, error) {
	owned, err := linkRepo.FindUserPlayer(tx, userID)
	if err != nil {
		return nil, err
	}
	if owned != nil {
		return nil, entity.ErrAccountAlreadyLinked
	}

	var player *entity.Player
	switch selection.Mode {
	case PlayerSelectionExisting:
		player, err = playerRepo.GetByID(tx, selection.PlayerID)
		if err != nil {
			return nil, err
		}
	case PlayerSelectionNew:
		player, err = entity.NewPlayer(playerIDGen.New(), strings.TrimSpace(selection.Name))
		if err != nil {
			return nil, err
		}
		if err := playerRepo.Create(tx, player); err != nil {
			return nil, err
		}
	default:
		return nil, entity.ErrInvalidPlayerSelection
	}

	if err := linkRepo.LinkPlayer(tx, userID, player.ID()); err != nil {
		return nil, err
	}
	return &PlayerDTO{ID: player.ID(), Name: player.Name()}, nil
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

func (uc *UserPlayerLinksUseCase) ListUnlinkedPlayers(ctx context.Context, q ListUnlinkedPlayersQuery) ([]AvailablePlayerDTO, error) {
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

	var result []AvailablePlayerDTO
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
