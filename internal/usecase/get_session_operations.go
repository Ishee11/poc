package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type GetSessionOperationsQuery struct {
	SessionID entity.SessionID

	Limit  int
	Offset int
}

type OperationDTO struct {
	ID          entity.OperationID
	Type        entity.OperationType
	PlayerID    entity.PlayerID
	Chips       int64
	CreatedAt   time.Time
	ReferenceID *entity.OperationID
}

type GetSessionOperationsResponse struct {
	Operations []OperationDTO
}

// --- usecase ---

type GetSessionOperationsUseCase struct {
	sessionReader SessionReader
	opReader      OperationListReader
	txManager     TxManager
}

func NewGetSessionOperationsUseCase(
	sessionReader SessionReader,
	opReader OperationListReader,
	txManager TxManager,
) *GetSessionOperationsUseCase {
	return &GetSessionOperationsUseCase{
		sessionReader: sessionReader,
		opReader:      opReader,
		txManager:     txManager,
	}
}

func (uc *GetSessionOperationsUseCase) Execute(
	q GetSessionOperationsQuery,
) (*GetSessionOperationsResponse, error) {

	var result *GetSessionOperationsResponse

	err := uc.txManager.RunInTx(func(tx Tx) error {

		// 1. проверяем что session существует
		_, err := uc.sessionReader.FindByID(tx, q.SessionID)
		if err != nil {
			return err
		}

		// 2. читаем операции
		ops, err := uc.opReader.ListBySession(tx, q.SessionID, q.Limit, q.Offset)
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

		result = &GetSessionOperationsResponse{
			Operations: res,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
