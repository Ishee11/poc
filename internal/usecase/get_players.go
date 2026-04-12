package usecase

import (
	"github.com/ishee11/poc/internal/entity"
)

type GetPlayersUseCase struct {
	repo      PlayerRepository
	txManager TxManager
}

type PlayerDTO struct {
	ID   entity.PlayerID
	Name string
}

func NewGetPlayersUseCase(
	repo PlayerRepository,
	txManager TxManager,
) *GetPlayersUseCase {
	return &GetPlayersUseCase{
		repo:      repo,
		txManager: txManager,
	}
}

func (uc *GetPlayersUseCase) Execute() ([]PlayerDTO, error) {

	var result []PlayerDTO

	err := uc.txManager.RunInTx(func(tx Tx) error {

		players, err := uc.repo.List(tx)
		if err != nil {
			return err
		}

		for _, p := range players {
			result = append(result, PlayerDTO{
				ID:   p.ID(),
				Name: p.Name(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
