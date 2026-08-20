package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/ishee11/poc/internal/entity"
)

const minRegisterPasswordLength = 12

type RegisterUserCommand struct {
	Email    string
	Password string
	Player   PlayerSelection
}

type RegisterUserUseCase struct {
	userRepo    AuthUserRepository
	txManager   TxManager
	idGen       AuthUserIDGenerator
	passwords   PasswordHasher
	clock       Clock
	linkRepo    UserPlayerLinkRepository
	playerRepo  PlayerRepository
	playerIDGen PlayerIDGenerator
}

func NewRegisterUserUseCase(
	userRepo AuthUserRepository,
	txManager TxManager,
	idGen AuthUserIDGenerator,
	passwords PasswordHasher,
	clock Clock,
	linkRepo UserPlayerLinkRepository,
	playerRepo PlayerRepository,
	playerIDGen PlayerIDGenerator,
) *RegisterUserUseCase {
	if clock == nil {
		clock = SystemClock{}
	}

	return &RegisterUserUseCase{
		userRepo:    userRepo,
		txManager:   txManager,
		idGen:       idGen,
		passwords:   passwords,
		clock:       clock,
		linkRepo:    linkRepo,
		playerRepo:  playerRepo,
		playerIDGen: playerIDGen,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, cmd RegisterUserCommand) (*PlayerDTO, error) {
	email := strings.TrimSpace(cmd.Email)
	if email == "" {
		return nil, entity.ErrInvalidAuthEmail
	}
	if len(cmd.Password) < minRegisterPasswordLength {
		return nil, entity.ErrPasswordTooShort
	}
	if err := cmd.Player.Validate(); err != nil {
		return nil, err
	}

	var registeredPlayer *PlayerDTO
	err := uc.txManager.RunInTx(ctx, func(tx Tx) error {
		_, err := uc.userRepo.FindUserByEmail(tx, email)
		if err == nil {
			return entity.ErrAuthUserAlreadyExists
		}
		if !errors.Is(err, entity.ErrAuthUserNotFound) {
			return err
		}

		passwordHash, err := uc.passwords.HashPassword(cmd.Password)
		if err != nil {
			return err
		}

		user, err := entity.NewAuthUser(
			uc.idGen.New(),
			email,
			passwordHash,
			entity.AuthRoleUser,
			uc.clock.Now(),
		)
		if err != nil {
			return err
		}

		if err := uc.userRepo.Save(tx, user); err != nil {
			return err
		}

		registeredPlayer, err = chooseOrCreatePlayerInTx(
			tx,
			user.ID,
			cmd.Player,
			uc.linkRepo,
			uc.playerRepo,
			uc.playerIDGen,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return registeredPlayer, nil
}
