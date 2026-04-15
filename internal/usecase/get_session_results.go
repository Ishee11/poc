package usecase

type GetSessionResultsUseCase struct {
	sessionReader SessionReader
	projection    ProjectionRepository
	txManager     TxManager
}

func NewGetSessionResultsUseCase(
	sessionReader SessionReader,
	projection ProjectionRepository,
	txManager TxManager,
) *GetSessionResultsUseCase {
	return &GetSessionResultsUseCase{
		sessionReader: sessionReader,
		projection:    projection,
		txManager:     txManager,
	}
}

func (uc *GetSessionResultsUseCase) Execute(
	q GetSessionResultsQuery,
) ([]PlayerResultDTO, error) {

	var result []PlayerResultDTO

	err := uc.txManager.RunInTx(func(tx Tx) error {

		// 1. загрузка session
		session, err := uc.sessionReader.FindByID(tx, q.SessionID)
		if err != nil {
			return err
		}

		// 2. агрегаты по игрокам
		playerAggs, err := uc.projection.GetPlayerAggregates(tx, q.SessionID)
		if err != nil {
			return err
		}

		// 3. сбор результата
		res := make([]PlayerResultDTO, 0, len(playerAggs))

		rate := session.ChipRate().Value() // :contentReference[oaicite:1]{index=1}

		for playerID, aggr := range playerAggs {

			profitChips := aggr.CashOut - aggr.BuyIn

			var profitMoney int64
			if rate > 0 {
				profitMoney = profitChips / rate
			}

			res = append(res, PlayerResultDTO{
				PlayerID:     playerID,
				BuyInChips:   aggr.BuyIn,
				CashOutChips: aggr.CashOut,
				ProfitChips:  profitChips,
				ProfitMoney:  profitMoney,
			})
		}

		result = res

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
