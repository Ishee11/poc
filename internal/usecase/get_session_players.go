package usecase

type GetSessionPlayersUseCase struct {
	playerRepo    PlayerRepository
	txManager     TxManager
	sessionReader SessionReader
}

func NewGetSessionPlayersUseCase(
	playerRepo PlayerRepository,
	txManager TxManager,
	sessionReader SessionReader,
) *GetSessionPlayersUseCase {
	return &GetSessionPlayersUseCase{
		playerRepo:    playerRepo,
		txManager:     txManager,
		sessionReader: sessionReader,
	}
}

func (uc *GetSessionPlayersUseCase) Execute(
	q GetSessionPlayersQuery,
) ([]PlayerDTO, error) {

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

func (uc *GetSessionPlayersUseCase) execute(
	tx Tx,
	q GetSessionPlayersQuery,
) ([]PlayerDTO, error) {

	// 1. проверка session
	if _, err := uc.sessionReader.FindByID(tx, q.SessionID); err != nil {
		return nil, err
	}

	// 2. получаем игроков
	return uc.playerRepo.ListBySession(tx, q.SessionID)
}
