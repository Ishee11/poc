package usecase

import "github.com/ishee11/poc/internal/entity"

type GetSessionPlayersQuery struct {
	SessionID entity.SessionID
}

type GetSessionPlayersUseCase struct {
	playerRepo PlayerRepository
	txManager  TxManager
}

func NewGetSessionPlayersUseCase(
	playerRepo PlayerRepository,
	txManager TxManager,
) *GetSessionPlayersUseCase {
	return &GetSessionPlayersUseCase{
		playerRepo: playerRepo,
		txManager:  txManager,
	}
}

func (uc *GetSessionPlayersUseCase) Execute(
	q GetSessionPlayersQuery,
) ([]PlayerDTO, error) {

	var result []PlayerDTO

	err := uc.txManager.RunInTx(func(tx Tx) error {
		players, err := uc.playerRepo.ListBySession(tx, q.SessionID)
		if err != nil {
			return err
		}
		result = players
		return nil
	})

	return result, err
}
