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
	if q.Limit <= 0 {
		q.Limit = 20
	}

	var result []PlayerStat

	err := uc.txManager.RunInTx(func(tx Tx) error {
		players, err := uc.statsRepo.ListPlayers(tx, PlayerStatsFilter{
			Limit: q.Limit,
			From:  q.From,
			To:    q.To,
		})
		if err != nil {
			return err
		}

		result = players

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
