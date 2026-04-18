package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

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

	// 1. проверка session (опционально)
	if _, err := uc.sessionReader.FindByID(tx, q.SessionID); err != nil {
		return nil, err
	}

	// 2. нормализация
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	// 3. читаем операции
	ops, err := uc.projection.ListBySession(tx, q.SessionID, limit, offset)
	if err != nil {
		return nil, err
	}

	// 4. маппинг
	result := make([]OperationDTO, 0, len(ops))
	for _, op := range ops {
		var referenceID *entity.OperationID
		if op.ReferenceID() != nil {
			ref := *op.ReferenceID()
			referenceID = &ref
		}

		result = append(result, OperationDTO{
			ID:          op.ID(),
			Type:        op.Type(),
			PlayerID:    op.PlayerID(),
			Chips:       op.Chips(),
			CreatedAt:   op.CreatedAt().Format(time.RFC3339),
			ReferenceID: referenceID,
		})
	}

	return result, nil
}
