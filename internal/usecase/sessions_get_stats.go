package usecase

type GetStatsSessionsUseCase struct {
	statsRepo StatsRepository
	txManager TxManager
}

func NewGetStatsSessionsUseCase(
	statsRepo StatsRepository,
	txManager TxManager,
) *GetStatsSessionsUseCase {
	return &GetStatsSessionsUseCase{
		statsRepo: statsRepo,
		txManager: txManager,
	}
}

func (uc *GetStatsSessionsUseCase) Execute(q GetStatsSessionsQuery) ([]SessionStat, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}

	var result []SessionStat

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

func (uc *GetStatsSessionsUseCase) execute(
	tx Tx,
	q GetStatsSessionsQuery,
) ([]SessionStat, error) {

	return uc.statsRepo.ListSessions(tx, SessionStatsFilter{
		Limit:        q.Limit,
		From:         q.From,
		To:           q.To,
		ViewerUserID: q.ViewerUserID,
	})
}
