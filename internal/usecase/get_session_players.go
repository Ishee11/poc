package usecase

import "github.com/ishee11/poc/internal/entity"

type GetSessionPlayersUseCase struct {
	projection    ProjectionRepository
	playerRepo    PlayerRepository
	txManager     TxManager
	sessionReader SessionReader
}

func NewGetSessionPlayersUseCase(
	projection ProjectionRepository,
	playerRepo PlayerRepository,
	txManager TxManager,
	sessionReader SessionReader,
) *GetSessionPlayersUseCase {
	return &GetSessionPlayersUseCase{
		projection:    projection,
		playerRepo:    playerRepo,
		txManager:     txManager,
		sessionReader: sessionReader,
	}
}

func (uc *GetSessionPlayersUseCase) Execute(
	q GetSessionPlayersQuery,
) ([]SessionPlayerDTO, error) {

	var result []SessionPlayerDTO

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

func (uc *GetSessionPlayersUseCase) execute(
	tx Tx,
	q GetSessionPlayersQuery,
) ([]SessionPlayerDTO, error) {

	session, err := uc.sessionReader.FindByID(tx, q.SessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, entity.ErrSessionNotFound
	}

	aggs, err := uc.projection.GetPlayerAggregates(tx, q.SessionID)
	if err != nil {
		return nil, err
	}

	result := make([]SessionPlayerDTO, 0, len(aggs))

	for playerID, agg := range aggs {
		inGame := session.Status() == entity.StatusActive && agg.BuyIn > agg.CashOut
		player, err := uc.playerRepo.GetByID(tx, playerID)
		if err != nil {
			return nil, err
		}
		profitChips := agg.CashOut - agg.BuyIn
		profitMoney := int64(0)
		if session.ChipRate().Value() > 0 {
			profitMoney = profitChips / session.ChipRate().Value()
		}

		result = append(result, SessionPlayerDTO{
			PlayerID:    playerID,
			Name:        player.Name(),
			BuyIn:       agg.BuyIn,
			CashOut:     agg.CashOut,
			ProfitChips: profitChips,
			ProfitMoney: profitMoney,
			InGame:      inGame,
		})
	}

	return result, nil

}
