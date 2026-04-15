package usecase

type GetSessionOperationsUseCase struct {
	sessionReader SessionReader
	projection    ProjectionRepository
	txManager     TxManager
}

func NewGetSessionOperationsUseCase(
	sessionReader SessionReader,
	projection ProjectionRepository,
	txManager TxManager,
) *GetSessionOperationsUseCase {
	return &GetSessionOperationsUseCase{
		sessionReader: sessionReader,
		projection:    projection,
		txManager:     txManager,
	}
}

func (uc *GetSessionOperationsUseCase) Execute(
	q GetSessionOperationsQuery,
) ([]OperationDTO, error) {

	var result []OperationDTO

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

func (uc *GetSessionOperationsUseCase) execute(
	tx Tx,
	q GetSessionOperationsQuery,
) ([]OperationDTO, error) {

	// 1. проверяем session
	if _, err := uc.sessionReader.FindByID(tx, q.SessionID); err != nil {
		return nil, err
	}

	// 2. читаем операции
	ops, err := uc.projection.ListBySession(tx, q.SessionID, q.Limit, q.Offset)
	if err != nil {
		return nil, err
	}

	// 3. маппинг
	result := make([]OperationDTO, 0, len(ops))
	for _, op := range ops {
		result = append(result, OperationDTO{
			ID:          op.ID(),
			Type:        op.Type(),
			PlayerID:    op.PlayerID(),
			Chips:       op.Chips(),
			CreatedAt:   op.CreatedAt(),
			ReferenceID: op.ReferenceID(),
		})
	}

	return result, nil
}
