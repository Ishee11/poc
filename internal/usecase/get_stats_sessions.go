package usecase

type GetStatsSessionsQuery struct {
	Limit int
	From  *DateTimeRangeBound
	To    *DateTimeRangeBound
}

type GetStatsSessionsResponse struct {
	Sessions []SessionStat
}

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

func (uc *GetStatsSessionsUseCase) Execute(q GetStatsSessionsQuery) (*GetStatsSessionsResponse, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}

	var result *GetStatsSessionsResponse

	err := uc.txManager.RunInTx(func(tx Tx) error {
		sessions, err := uc.statsRepo.ListSessions(tx, SessionStatsFilter{
			Limit: q.Limit,
			From:  q.From,
			To:    q.To,
		})
		if err != nil {
			return err
		}

		result = &GetStatsSessionsResponse{
			Sessions: sessions,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
