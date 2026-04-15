package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type GetSessionResponse struct {
	SessionID    entity.SessionID
	Status       entity.Status
	ChipRate     int64
	CreatedAt    string
	TotalBuyIn   int64
	TotalCashOut int64
	TotalChips   int64
}

type GetSessionUseCase struct {
	sessionReader SessionReader
	projection    ProjectionRepository
	txManager     TxManager
}

func NewGetSessionUseCase(
	sessionReader SessionReader,
	projection ProjectionRepository,
	txManager TxManager,
) *GetSessionUseCase {
	return &GetSessionUseCase{
		sessionReader: sessionReader,
		projection:    projection,
		txManager:     txManager,
	}
}

func (uc *GetSessionUseCase) Execute(q GetSessionQuery) (*GetSessionResponse, error) {
	var result *GetSessionResponse

	err := uc.txManager.RunInTx(func(tx Tx) error {
		// 1. загрузка session
		session, err := uc.sessionReader.FindByID(tx, q.SessionID)
		if err != nil {
			return err
		}

		// 2. агрегаты (источник истины)
		aggr, err := uc.projection.GetSessionAggregates(tx, q.SessionID)
		if err != nil {
			return err
		}

		// 3. сбор ответа
		result = &GetSessionResponse{
			SessionID:    session.ID(),
			Status:       session.Status(),
			ChipRate:     session.ChipRate().Value(),
			CreatedAt:    session.CreatedAt().Format(time.RFC3339),
			TotalBuyIn:   aggr.TotalBuyIn,
			TotalCashOut: aggr.TotalCashOut,
			TotalChips:   aggr.TotalBuyIn - aggr.TotalCashOut,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
