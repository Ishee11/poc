package usecase

import (
	"errors"
	"strings"

	"github.com/ishee11/poc/internal/entity"
)

const minSeedAdminPasswordLength = 12

type SeedAdminCommand struct {
	Email    string
	Password string
}

type SeedAdminUseCase struct {
	userRepo  AuthUserRepository
	txManager TxManager
	idGen     AuthUserIDGenerator
	passwords PasswordHasher
	clock     Clock
}

func NewSeedAdminUseCase(
	userRepo AuthUserRepository,
	txManager TxManager,
	idGen AuthUserIDGenerator,
	passwords PasswordHasher,
	clock Clock,
) *SeedAdminUseCase {
	if clock == nil {
		clock = SystemClock{}
	}

	return &SeedAdminUseCase{
		userRepo:  userRepo,
		txManager: txManager,
		idGen:     idGen,
		passwords: passwords,
		clock:     clock,
	}
}

func (uc *SeedAdminUseCase) Execute(cmd SeedAdminCommand) error {
	email := strings.TrimSpace(cmd.Email)
	if email == "" && cmd.Password == "" {
		return nil
	}
	if email == "" {
		return entity.ErrInvalidAuthEmail
	}
	if len(cmd.Password) < minSeedAdminPasswordLength {
		return entity.ErrPasswordTooShort
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		_, err := uc.userRepo.FindUserByEmail(tx, email)
		if err == nil {
			return nil
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
			entity.AuthRoleAdmin,
			uc.clock.Now(),
		)
		if err != nil {
			return err
		}

		return uc.userRepo.Save(tx, user)
	})
}
