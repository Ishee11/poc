package usecase

import "github.com/ishee11/poc/internal/entity"

type GetSessionResultsUseCase struct {
	sessionReader SessionReader
	projection    ProjectionRepository
	playerReader  PlayerReader
	txManager     TxManager
}

func NewGetSessionResultsUseCase(
	sessionReader SessionReader,
	projection ProjectionRepository,
	playerReader PlayerReader,
	txManager TxManager,
) *GetSessionResultsUseCase {
	return &GetSessionResultsUseCase{
		sessionReader: sessionReader,
		projection:    projection,
		playerReader:  playerReader,
		txManager:     txManager,
	}
}

func (uc *GetSessionResultsUseCase) Execute(
	q GetSessionResultsQuery,
) ([]PlayerResultDTO, error) {

	var result []PlayerResultDTO

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

func (uc *GetSessionResultsUseCase) execute(
	tx Tx,
	q GetSessionResultsQuery,
) ([]PlayerResultDTO, error) {

	// 1. session
	session, err := uc.sessionReader.FindByID(tx, q.SessionID)
	if err != nil {
		return nil, err
	}

	// 2. агрегаты
	playerAggs, err := uc.projection.GetPlayerAggregates(tx, q.SessionID)
	if err != nil {
		return nil, err
	}

	// 2.1 получаем игроков (НОВОЕ)
	players, err := uc.playerReader.ListBySession(tx, q.SessionID)
	if err != nil {
		return nil, err
	}

	// строим map id → name
	playerNames := make(map[entity.PlayerID]string, len(players))
	for _, p := range players {
		playerNames[p.ID] = p.Name
	}

	// 3. сбор результата
	result := make([]PlayerResultDTO, 0, len(playerAggs))

	rate := session.ChipRate().Value()

	for playerID, aggr := range playerAggs {

		profitChips := aggr.CashOut - aggr.BuyIn

		var profitMoney int64
		if rate > 0 {
			profitMoney = profitChips / rate
		}

		name := playerNames[playerID]
		if name == "" {
			name = string(playerID) // fallback (опционально)
		}

		result = append(result, PlayerResultDTO{
			PlayerID:     playerID,
			PlayerName:   name, // 👈 ВОТ ЭТО
			BuyInChips:   aggr.BuyIn,
			CashOutChips: aggr.CashOut,
			ProfitChips:  profitChips,
			ProfitMoney:  profitMoney,
		})
	}

	return result, nil
}
