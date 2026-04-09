package usecase

import (
	"github.com/ishee11/poc/internal/entity"
)

type GetSessionQuery struct {
	SessionID entity.SessionID
}

type GetSessionResponse struct {
	SessionID    entity.SessionID
	Status       entity.Status
	ChipRate     int64
	TotalBuyIn   int64
	TotalCashOut int64
	TotalChips   int64
}

type GetSessionUseCase struct {
	sessionRepo SessionRepository
	opRepo      OperationRepository
	txManager   TxManager
}

func NewGetSessionUseCase(
	sessionRepo SessionRepository,
	opRepo OperationRepository,
	txManager TxManager,
) *GetSessionUseCase {
	return &GetSessionUseCase{
		sessionRepo: sessionRepo,
		opRepo:      opRepo,
		txManager:   txManager,
	}
}

func (uc *GetSessionUseCase) Execute(q GetSessionQuery) (*GetSessionResponse, error) {
	var result *GetSessionResponse

	err := uc.txManager.RunInTx(func(tx Tx) error {
		// 1. загрузка session
		session, err := uc.sessionRepo.FindByID(tx, q.SessionID)
		if err != nil {
			return err
		}

		// 2. агрегаты (источник истины)
		aggr, err := uc.opRepo.GetSessionAggregates(tx, q.SessionID)
		if err != nil {
			return err
		}

		// 3. сбор ответа
		result = &GetSessionResponse{
			SessionID:    session.ID(),
			Status:       session.Status(),
			ChipRate:     session.ChipRate().Value(),
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
