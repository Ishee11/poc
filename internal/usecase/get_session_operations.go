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

		// 1. проверяем что session существует
		_, err := uc.sessionReader.FindByID(tx, q.SessionID)
		if err != nil {
			return err
		}

		// 2. читаем операции
		ops, err := uc.projection.ListBySession(tx, q.SessionID, q.Limit, q.Offset)
		if err != nil {
			return err
		}

		// 3. маппинг
		res := make([]OperationDTO, 0, len(ops))

		for _, op := range ops {
			res = append(res, OperationDTO{
				ID:          op.ID(),
				Type:        op.Type(),
				PlayerID:    op.PlayerID(),
				Chips:       op.Chips(),
				CreatedAt:   op.CreatedAt(),
				ReferenceID: op.ReferenceID(),
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
