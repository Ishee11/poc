package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

// CashOut godoc
// @Summary Cash out
// @Description Withdraw chips from session
// @Tags operations
// @Accept json
// @Produce json
// @Param request body CashOutRequest true "Cash-out request"
// @Success 200 {object} OperationAcknowledgement
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /operations/cash-out [post]
func (h *OperationHandler) CashOut(w http.ResponseWriter, r *http.Request) {
	var req CashOutRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	ack, err := h.cashOutUC.Execute(r.Context(), command.CashOutCommand{
		RequestID: req.RequestID,
		SessionID: entity.SessionID(req.SessionID),
		PlayerID:  entity.PlayerID(req.PlayerID),
		Chips:     req.Chips,
	})

	if err != nil {
		logOperationCommandFailure(r, req.RequestID, "cash_out", req.SessionID, err)
		writeError(w, r, err)
		return
	}

	logOperationAcknowledgement(r, ack)
	writeJSON(w, http.StatusOK, ack)
}
