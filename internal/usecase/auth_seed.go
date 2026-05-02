package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/ishee11/poc/internal/entity"
)

const minSeedUserPasswordLength = 12

type SeedUserCommand struct {
	Email    string
	Password string
	Role     entity.AuthRole
}

type SeedUserUseCase struct {
	userRepo  AuthUserRepository
	txManager TxManager
	idGen     AuthUserIDGenerator
	passwords PasswordHasher
	clock     Clock
}

func NewSeedUserUseCase(
	userRepo AuthUserRepository,
	txManager TxManager,
	idGen AuthUserIDGenerator,
	passwords PasswordHasher,
	clock Clock,
) *SeedUserUseCase {
	if clock == nil {
		clock = SystemClock{}
	}

	return &SeedUserUseCase{
		userRepo:  userRepo,
		txManager: txManager,
		idGen:     idGen,
		passwords: passwords,
		clock:     clock,
	}
}

func (uc *SeedUserUseCase) Execute(ctx context.Context, cmd SeedUserCommand) error {
	email := strings.TrimSpace(cmd.Email)
	if email == "" && cmd.Password == "" {
		return nil
	}
	if email == "" {
		return entity.ErrInvalidAuthEmail
	}
	if len(cmd.Password) < minSeedUserPasswordLength {
		return entity.ErrPasswordTooShort
	}
	if !cmd.Role.Valid() {
		return entity.ErrInvalidAuthRole
	}

	return uc.txManager.RunInTx(ctx, func(tx Tx) error {
		existing, err := uc.userRepo.FindUserByEmail(tx, email)
		if err == nil {
			passwordHash, err := uc.passwords.HashPassword(cmd.Password)
			if err != nil {
				return err
			}

			existing.Email = email
			existing.PasswordHash = passwordHash
			existing.Role = cmd.Role
			existing.Status = entity.AuthUserStatusActive
			existing.UpdatedAt = uc.clock.Now()
			return uc.userRepo.Save(tx, existing)
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
			cmd.Role,
			uc.clock.Now(),
		)
		if err != nil {
			return err
		}

		return uc.userRepo.Save(tx, user)
	})
}
