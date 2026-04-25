package usecase

import (
	"errors"
	"strings"

	"github.com/ishee11/poc/internal/entity"
)

const minRegisterPasswordLength = 12

type RegisterUserCommand struct {
	Email    string
	Password string
}

type RegisterUserUseCase struct {
	userRepo  AuthUserRepository
	txManager TxManager
	idGen     AuthUserIDGenerator
	passwords PasswordHasher
	clock     Clock
}

func NewRegisterUserUseCase(
	userRepo AuthUserRepository,
	txManager TxManager,
	idGen AuthUserIDGenerator,
	passwords PasswordHasher,
	clock Clock,
) *RegisterUserUseCase {
	if clock == nil {
		clock = SystemClock{}
	}

	return &RegisterUserUseCase{
		userRepo:  userRepo,
		txManager: txManager,
		idGen:     idGen,
		passwords: passwords,
		clock:     clock,
	}
}

func (uc *RegisterUserUseCase) Execute(cmd RegisterUserCommand) error {
	email := strings.TrimSpace(cmd.Email)
	if email == "" {
		return entity.ErrInvalidAuthEmail
	}
	if len(cmd.Password) < minRegisterPasswordLength {
		return entity.ErrPasswordTooShort
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
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

		return uc.userRepo.Save(tx, user)
	})
}
