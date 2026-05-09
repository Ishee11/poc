package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/ishee11/poc/internal/entity"
)

var (
	ErrInvalidExpense        = errors.New("invalid expense")
	ErrInvalidExpensePayment = errors.New("invalid expense payment")
)

type SessionExpense struct {
	ID           string                  `json:"id"`
	SessionID    entity.SessionID        `json:"session_id"`
	Title        string                  `json:"title"`
	Amount       int64                   `json:"amount"`
	Participants []entity.PlayerID       `json:"participants"`
	Payments     []SessionExpensePayment `json:"payments"`
	CreatedAt    string                  `json:"created_at"`
}

type SessionExpensePayment struct {
	PlayerID entity.PlayerID `json:"player_id"`
	Amount   int64           `json:"amount"`
}

type CreateSessionExpenseInput struct {
	SessionID               entity.SessionID
	Title                   string
	Amount                  int64
	Participants            []entity.PlayerID
	Payments                []SessionExpensePayment
	AllowClosedModification bool
}

type DeleteSessionExpenseInput struct {
	ExpenseID               string
	AllowClosedModification bool
}

type SessionExpenseRepository interface {
	Create(tx Tx, expense SessionExpense) error
	FindByID(tx Tx, expenseID string) (*SessionExpense, error)
	ListBySession(tx Tx, sessionID entity.SessionID) ([]SessionExpense, error)
	Delete(tx Tx, expenseID string) error
}

type ExpenseIDGenerator interface {
	New() string
}

type SessionExpenseService struct {
	repo          SessionExpenseRepository
	sessionReader SessionReader
	sessionWriter SessionWriter
	txManager     TxManager
	idGen         ExpenseIDGenerator
}

func NewSessionExpenseService(
	repo SessionExpenseRepository,
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	txManager TxManager,
	idGen ExpenseIDGenerator,
) *SessionExpenseService {
	return &SessionExpenseService{
		repo:          repo,
		sessionReader: sessionReader,
		sessionWriter: sessionWriter,
		txManager:     txManager,
		idGen:         idGen,
	}
}

func (s *SessionExpenseService) Create(ctx context.Context, input CreateSessionExpenseInput) (SessionExpense, error) {
	var created SessionExpense
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		session, err := s.sessionReader.FindByID(tx, input.SessionID)
		if err != nil {
			return err
		}
		if session.ExpensesClosed() && !input.AllowClosedModification {
			return entity.ErrSessionExpensesClosed
		}

		expense, err := s.buildExpense(input)
		if err != nil {
			return err
		}

		if err := s.repo.Create(tx, expense); err != nil {
			return err
		}

		created = expense
		return nil
	})
	return created, err
}

func (s *SessionExpenseService) List(ctx context.Context, sessionID entity.SessionID) ([]SessionExpense, error) {
	var expenses []SessionExpense
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		if _, err := s.sessionReader.FindByID(tx, sessionID); err != nil {
			return err
		}

		var err error
		expenses, err = s.repo.ListBySession(tx, sessionID)
		return err
	})
	return expenses, err
}

func (s *SessionExpenseService) Delete(ctx context.Context, input DeleteSessionExpenseInput) error {
	return s.txManager.RunInTx(ctx, func(tx Tx) error {
		expense, err := s.repo.FindByID(tx, input.ExpenseID)
		if err != nil {
			return err
		}
		if expense == nil {
			return nil
		}

		session, err := s.sessionReader.FindByID(tx, expense.SessionID)
		if err != nil {
			return err
		}
		if session.ExpensesClosed() && !input.AllowClosedModification {
			return entity.ErrSessionExpensesClosed
		}

		return s.repo.Delete(tx, input.ExpenseID)
	})
}

func (s *SessionExpenseService) Close(ctx context.Context, sessionID entity.SessionID) error {
	return s.txManager.RunInTx(ctx, func(tx Tx) error {
		session, err := s.sessionReader.FindByID(tx, sessionID)
		if err != nil {
			return err
		}

		session.CloseExpenses()
		return s.sessionWriter.Save(tx, session)
	})
}

func (s *SessionExpenseService) buildExpense(input CreateSessionExpenseInput) (SessionExpense, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" || input.Amount <= 0 || len(input.Participants) == 0 || len(input.Payments) == 0 {
		return SessionExpense{}, ErrInvalidExpense
	}

	participants := uniquePlayers(input.Participants)
	if len(participants) == 0 {
		return SessionExpense{}, ErrInvalidExpense
	}

	payments := make([]SessionExpensePayment, 0, len(input.Payments))
	var paidTotal int64
	seenPayers := make(map[entity.PlayerID]struct{}, len(input.Payments))
	for _, payment := range input.Payments {
		if payment.PlayerID == "" || payment.Amount <= 0 {
			return SessionExpense{}, ErrInvalidExpensePayment
		}
		if _, ok := seenPayers[payment.PlayerID]; ok {
			return SessionExpense{}, ErrInvalidExpensePayment
		}
		seenPayers[payment.PlayerID] = struct{}{}
		paidTotal += payment.Amount
		payments = append(payments, payment)
	}
	if paidTotal != input.Amount {
		return SessionExpense{}, ErrInvalidExpensePayment
	}

	return SessionExpense{
		ID:           s.idGen.New(),
		SessionID:    input.SessionID,
		Title:        title,
		Amount:       input.Amount,
		Participants: participants,
		Payments:     payments,
	}, nil
}

func uniquePlayers(players []entity.PlayerID) []entity.PlayerID {
	result := make([]entity.PlayerID, 0, len(players))
	seen := make(map[entity.PlayerID]struct{}, len(players))
	for _, playerID := range players {
		if playerID == "" {
			continue
		}
		if _, ok := seen[playerID]; ok {
			continue
		}
		seen[playerID] = struct{}{}
		result = append(result, playerID)
	}
	return result
}
