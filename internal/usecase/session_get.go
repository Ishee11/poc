package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type GetSessionResponse struct {
	SessionID    entity.SessionID `json:"session_id"`
	Status       entity.Status    `json:"status"`
	ChipRate     int64            `json:"chip_rate"`
	BigBlind     int64            `json:"big_blind"`
	CreatedAt    string           `json:"created_at"`
	FinishedAt   *string          `json:"finished_at,omitempty"`
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
	session, err := uc.sessionReader.FindByID(tx, q.SessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, entity.ErrSessionNotFound
	}

	var finishedAt *string
	if session.FinishedAt() != nil {
		formatted := session.FinishedAt().Format(time.RFC3339)
		finishedAt = &formatted
	}

	return &GetSessionResponse{
		SessionID:    session.ID(),
		Status:       session.Status(),
		ChipRate:     session.ChipRate().Value(),
		BigBlind:     session.BigBlind(),
		CreatedAt:    session.CreatedAt().Format(time.RFC3339),
		FinishedAt:   finishedAt,
		TotalBuyIn:   session.TotalBuyIn(),
		TotalCashOut: session.TotalCashOut(),
		TotalChips:   session.TotalChips(),
	}, nil
}
