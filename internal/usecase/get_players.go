package usecase

type GetPlayersUseCase struct {
	playerRepo PlayerRepository
	txManager  TxManager
}

func NewGetPlayersUseCase(
	playerRepo PlayerRepository,
	txManager TxManager,
) *GetPlayersUseCase {
	return &GetPlayersUseCase{
		playerRepo: playerRepo,
		txManager:  txManager,
	}
}

func (uc *GetPlayersUseCase) Execute(q GetPlayersQuery) ([]PlayerDTO, error) {
	var result []PlayerDTO

	err := uc.txManager.RunInTx(func(tx Tx) error {
		var err error
		result, err = uc.execute(tx, q)
		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *GetPlayersUseCase) execute(tx Tx, q GetPlayersQuery) ([]PlayerDTO, error) {
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

	return uc.playerRepo.List(tx, limit, offset)
}
