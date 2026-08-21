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
	if sessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}
	if !h.access.requireView(w, r, sessionID) {
		return
	}
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
	if !h.access.requireView(w, r, entity.SessionID(req.SessionID)) {
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
		SessionID:               entity.SessionID(req.SessionID),
		Title:                   req.Title,
		Amount:                  req.Amount,
		Participants:            participants,
		Payments:                payments,
		AllowClosedModification: h.isAdmin(r),
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
	sessionID := entity.SessionID(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}
	if !h.access.requireView(w, r, sessionID) {
		return
	}

	if err := h.expenseService.Delete(r.Context(), usecase.DeleteSessionExpenseInput{
		ExpenseID:               expenseID,
		AllowClosedModification: h.isAdmin(r),
	}); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *OperationHandler) CloseExpenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	defer r.Body.Close()

	var req CloseExpensesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}
	if req.SessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}
	if !h.access.requireView(w, r, entity.SessionID(req.SessionID)) {
		return
	}

	if err := h.expenseService.Close(r.Context(), entity.SessionID(req.SessionID)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OperationHandler) isAdmin(r *http.Request) bool {
	if h.authUC == nil {
		return false
	}

	cookie, err := r.Cookie(h.cookie.Name)
	if err != nil || cookie.Value == "" {
		return false
	}

	principal, err := h.authUC.CurrentUser(r.Context(), cookie.Value)
	if err != nil {
		return false
	}

	return principal.Role == entity.AuthRoleAdmin
}
