package usecase

import (
	"context"
	"errors"

	"github.com/ishee11/poc/internal/entity"
)

var ErrInvalidSettlementTransfer = errors.New("invalid settlement transfer")

type SettlementTransfer struct {
	ID        string           `json:"id"`
	SessionID entity.SessionID `json:"session_id"`
	From      entity.PlayerID  `json:"from"`
	To        entity.PlayerID  `json:"to"`
	Amount    int64            `json:"amount"`
	Position  int              `json:"position"`
}

type SaveSettlementTransfersInput struct {
	SessionID entity.SessionID
	Transfers []SettlementTransfer
}

type SettlementTransferRepository interface {
	ListBySession(tx Tx, sessionID entity.SessionID) ([]SettlementTransfer, error)
	ReplaceBySession(tx Tx, sessionID entity.SessionID, transfers []SettlementTransfer) error
}

type SettlementTransferIDGenerator interface {
	New() string
}

type SettlementTransferService struct {
	repo          SettlementTransferRepository
	sessionReader SessionReader
	txManager     TxManager
	idGen         SettlementTransferIDGenerator
}

func NewSettlementTransferService(
	repo SettlementTransferRepository,
	sessionReader SessionReader,
	txManager TxManager,
	idGen SettlementTransferIDGenerator,
) *SettlementTransferService {
	return &SettlementTransferService{
		repo:          repo,
		sessionReader: sessionReader,
		txManager:     txManager,
		idGen:         idGen,
	}
}

func (s *SettlementTransferService) List(ctx context.Context, sessionID entity.SessionID) ([]SettlementTransfer, error) {
	var transfers []SettlementTransfer
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		if _, err := s.sessionReader.FindByID(tx, sessionID); err != nil {
			return err
		}

		var err error
		transfers, err = s.repo.ListBySession(tx, sessionID)
		return err
	})
	return transfers, err
}

func (s *SettlementTransferService) Save(ctx context.Context, input SaveSettlementTransfersInput) ([]SettlementTransfer, error) {
	transfers, err := s.buildTransfers(input)
	if err != nil {
		return nil, err
	}

	err = s.txManager.RunInTx(ctx, func(tx Tx) error {
		if _, err := s.sessionReader.FindByID(tx, input.SessionID); err != nil {
			return err
		}

		return s.repo.ReplaceBySession(tx, input.SessionID, transfers)
	})
	return transfers, err
}

func (s *SettlementTransferService) buildTransfers(input SaveSettlementTransfersInput) ([]SettlementTransfer, error) {
	transfers := make([]SettlementTransfer, 0, len(input.Transfers))
	for idx, transfer := range input.Transfers {
		if transfer.From == "" || transfer.To == "" || transfer.From == transfer.To || transfer.Amount <= 0 {
			return nil, ErrInvalidSettlementTransfer
		}

		id := transfer.ID
		if id == "" {
			id = s.idGen.New()
		}

		transfers = append(transfers, SettlementTransfer{
			ID:        id,
			SessionID: input.SessionID,
			From:      transfer.From,
			To:        transfer.To,
			Amount:    transfer.Amount,
			Position:  idx,
		})
	}
	return transfers, nil
}
