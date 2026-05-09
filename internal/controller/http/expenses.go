package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *OperationHandler) Expenses(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListExpenses(w, r)
	case http.MethodPost:
		h.CreateExpense(w, r)
	case http.MethodDelete:
		h.DeleteExpense(w, r)
	default:
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
	}
}

func (h *OperationHandler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	sessionID := entity.SessionID(r.URL.Query().Get("session_id"))
	expenses, err := h.expenseService.List(r.Context(), sessionID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, expenses)
}

func (h *OperationHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var req CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	participants := make([]entity.PlayerID, 0, len(req.Participants))
	for _, playerID := range req.Participants {
		participants = append(participants, entity.PlayerID(playerID))
	}

	payments := make([]usecase.SessionExpensePayment, 0, len(req.Payments))
	for _, payment := range req.Payments {
		payments = append(payments, usecase.SessionExpensePayment{
			PlayerID: entity.PlayerID(payment.PlayerID),
			Amount:   payment.Amount,
		})
	}

	expense, err := h.expenseService.Create(r.Context(), usecase.CreateSessionExpenseInput{
		SessionID:    entity.SessionID(req.SessionID),
		Title:        req.Title,
		Amount:       req.Amount,
		Participants: participants,
		Payments:     payments,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, expense)
}

func (h *OperationHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	expenseID := r.URL.Query().Get("expense_id")
	if expenseID == "" {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	if err := h.expenseService.Delete(r.Context(), expenseID); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
