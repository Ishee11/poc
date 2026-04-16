package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type GetSessionResponse struct {
	SessionID    entity.SessionID `json:"session_id"`
	Status       entity.Status    `json:"status"`
	ChipRate     int64            `json:"chip_rate"`
	CreatedAt    string           `json:"created_at"`
	TotalBuyIn   int64            `json:"total_buy_in"`
	TotalCashOut int64            `json:"total_cash_out"`
	TotalChips   int64            `json:"total_chips"`
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
		var err error
		result, err = uc.execute(tx, q)
		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *GetSessionUseCase) execute(tx Tx, q GetSessionQuery) (*GetSessionResponse, error) {
	// 1. session
	session, err := uc.sessionReader.FindByID(tx, q.SessionID)
	if err != nil {
		return nil, err
	}

	// 2. агрегаты
	aggr, err := uc.projection.GetSessionAggregates(tx, q.SessionID)
	if err != nil {
		return nil, err
	}

	// 3. response
	return &GetSessionResponse{
		SessionID:    session.ID(),
		Status:       session.Status(),
		ChipRate:     session.ChipRate().Value(),
		CreatedAt:    session.CreatedAt().Format(time.RFC3339),
		TotalBuyIn:   aggr.TotalBuyIn,
		TotalCashOut: aggr.TotalCashOut,
		TotalChips:   aggr.TotalBuyIn - aggr.TotalCashOut,
	}, nil
}
