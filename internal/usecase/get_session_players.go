package usecase

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

	if _, err := uc.sessionReader.FindByID(tx, q.SessionID); err != nil {
		return nil, err
	}

	aggs, err := uc.projection.GetPlayerAggregates(tx, q.SessionID)
	if err != nil {
		return nil, err
	}

	result := make([]SessionPlayerDTO, 0, len(aggs))

	for playerID, agg := range aggs {
		inGame := agg.BuyIn > agg.CashOut

		result = append(result, SessionPlayerDTO{
			PlayerID: playerID,
			Name:     "", // TODO: через playerRepo.GetByID
			BuyIn:    agg.BuyIn,
			CashOut:  agg.CashOut,
			InGame:   inGame,
		})
	}

	return result, nil

}
