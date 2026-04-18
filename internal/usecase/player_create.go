package usecase

import (
	"strings"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type CreatePlayerUseCase struct {
	helper          *Helper
	txManager       TxManager
	idempotencyRepo IdempotencyRepository
}

func NewCreatePlayerUseCase(
	helper *Helper,
	txManager TxManager,
	idempotencyRepo IdempotencyRepository,
) *CreatePlayerUseCase {
	return &CreatePlayerUseCase{
		helper:          helper,
		txManager:       txManager,
		idempotencyRepo: idempotencyRepo,
	}
}

func (uc *CreatePlayerUseCase) Execute(cmd command.CreatePlayerCommand) (entity.PlayerID, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return "", entity.ErrInvalidPlayerName
	}

	var result entity.PlayerID

	err := uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {
			id, err := uc.execute(tx, name)
			if err != nil {
				return err
			}
			result = id
			return nil
		})
	})

	if err != nil {
		return "", err
	}

	return result, nil
}

func (uc *CreatePlayerUseCase) execute(tx Tx, name string) (entity.PlayerID, error) {
	player, err := uc.helper.BuildPlayer(name)
	if err != nil {
		return "", err
	}

	return uc.helper.SavePlayer(tx, player)
}
