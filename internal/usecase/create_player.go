package usecase

import (
	"github.com/ishee11/poc/internal/entity"
)

type PlayerIDGenerator interface {
	New() entity.PlayerID
}

type CreatePlayerUseCase struct {
	repo      PlayerRepository
	txManager TxManager
	idGen     PlayerIDGenerator
}

type CreatePlayerCommand struct {
	Name string
}

type CreatePlayerResponse struct {
	ID   entity.PlayerID
	Name string
}

func NewCreatePlayerUseCase(
	repo PlayerRepository,
	txManager TxManager,
	idGen PlayerIDGenerator,
) *CreatePlayerUseCase {
	return &CreatePlayerUseCase{
		repo:      repo,
		txManager: txManager,
		idGen:     idGen,
	}
}

func (uc *CreatePlayerUseCase) Execute(
	cmd CreatePlayerCommand,
) (*CreatePlayerResponse, error) {

	if cmd.Name == "" {
		return nil, entity.ErrInvalidPlayerName
	}

	var result *CreatePlayerResponse

	err := uc.txManager.RunInTx(func(tx Tx) error {

		id := uc.idGen.New()

		player, err := entity.NewPlayer(id, cmd.Name)
		if err != nil {
			return err
		}

		if err := uc.repo.Save(tx, player); err != nil {
			return err
		}

		result = &CreatePlayerResponse{
			ID:   player.ID(),
			Name: player.Name(),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
