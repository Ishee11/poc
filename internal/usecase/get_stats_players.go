package usecase

type GetStatsPlayersUseCase struct {
	statsRepo StatsRepository
	txManager TxManager
}

func NewGetStatsPlayersUseCase(
	statsRepo StatsRepository,
	txManager TxManager,
) *GetStatsPlayersUseCase {
	return &GetStatsPlayersUseCase{
		statsRepo: statsRepo,
		txManager: txManager,
	}
}

func (uc *GetStatsPlayersUseCase) Execute(q GetStatsPlayersQuery) ([]PlayerStat, error) {
	var result []PlayerStat

	err := uc.txManager.RunInTx(func(tx Tx) error {
		res, err := uc.execute(tx, q)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *GetStatsPlayersUseCase) execute(
	tx Tx,
	q GetStatsPlayersQuery,
) ([]PlayerStat, error) {

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return uc.statsRepo.ListPlayers(tx, PlayerStatsFilter{
		Limit: limit,
		From:  q.From,
		To:    q.To,
	})
}
