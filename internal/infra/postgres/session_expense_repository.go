package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type SessionExpenseRepository struct{}

func NewSessionExpenseRepository() *SessionExpenseRepository {
	return &SessionExpenseRepository{}
}

func (r *SessionExpenseRepository) Create(tx usecase.Tx, expense usecase.SessionExpense) error {
	if _, err := tx.Exec(context.Background(), `
		INSERT INTO session_expenses (id, session_id, title, amount)
		VALUES ($1, $2, $3, $4)
	`, expense.ID, expense.SessionID, expense.Title, expense.Amount); err != nil {
		return err
	}

	for _, playerID := range expense.Participants {
		if _, err := tx.Exec(context.Background(), `
			INSERT INTO session_expense_participants (expense_id, player_id)
			VALUES ($1, $2)
		`, expense.ID, playerID); err != nil {
			return err
		}
	}

	for _, payment := range expense.Payments {
		if _, err := tx.Exec(context.Background(), `
			INSERT INTO session_expense_payments (expense_id, player_id, amount)
			VALUES ($1, $2, $3)
		`, expense.ID, payment.PlayerID, payment.Amount); err != nil {
			return err
		}
	}

	return nil
}

func (r *SessionExpenseRepository) ListBySession(tx usecase.Tx, sessionID entity.SessionID) ([]usecase.SessionExpense, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT id, session_id, title, amount, created_at
		FROM session_expenses
		WHERE session_id = $1
		ORDER BY created_at DESC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenses := make([]usecase.SessionExpense, 0)
	for rows.Next() {
		var expense usecase.SessionExpense
		var createdAt time.Time
		if err := rows.Scan(&expense.ID, &expense.SessionID, &expense.Title, &expense.Amount, &createdAt); err != nil {
			return nil, err
		}
		expense.CreatedAt = createdAt.Format(time.RFC3339)
		expenses = append(expenses, expense)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for idx := range expenses {
		participants, err := r.listParticipants(tx, expenses[idx].ID)
		if err != nil {
			return nil, err
		}
		payments, err := r.listPayments(tx, expenses[idx].ID)
		if err != nil {
			return nil, err
		}
		expenses[idx].Participants = participants
		expenses[idx].Payments = payments
	}

	return expenses, nil
}

func (r *SessionExpenseRepository) Delete(tx usecase.Tx, expenseID string) error {
	_, err := tx.Exec(context.Background(), `DELETE FROM session_expenses WHERE id = $1`, expenseID)
	return err
}

func (r *SessionExpenseRepository) listParticipants(tx usecase.Tx, expenseID string) ([]entity.PlayerID, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT player_id
		FROM session_expense_participants
		WHERE expense_id = $1
		ORDER BY player_id
	`, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := make([]entity.PlayerID, 0)
	for rows.Next() {
		var playerID entity.PlayerID
		if err := rows.Scan(&playerID); err != nil {
			return nil, err
		}
		participants = append(participants, playerID)
	}
	return participants, rows.Err()
}

func (r *SessionExpenseRepository) listPayments(tx usecase.Tx, expenseID string) ([]usecase.SessionExpensePayment, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT player_id, amount
		FROM session_expense_payments
		WHERE expense_id = $1
		ORDER BY player_id
	`, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make([]usecase.SessionExpensePayment, 0)
	for rows.Next() {
		var payment usecase.SessionExpensePayment
		if err := rows.Scan(&payment.PlayerID, &payment.Amount); err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	return payments, nil
}
