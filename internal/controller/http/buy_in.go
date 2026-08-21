package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

// BuyIn godoc
// @Summary Buy-in
// @Description Add chips to session
// @Tags operations
// @Accept json
// @Produce json
// @Param request body BuyInRequest true "Buy-in request"
// @Success 200 {object} OperationAcknowledgement
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /operations/buy-in [post]
func (h *OperationHandler) BuyIn(w http.ResponseWriter, r *http.Request) {
	var req BuyInRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}
	if !h.access.requireView(w, r, entity.SessionID(req.SessionID)) {
		return
	}

	ack, err := h.buyInUC.Execute(r.Context(), command.BuyInCommand{
		RequestID: req.RequestID,
		SessionID: entity.SessionID(req.SessionID),
		PlayerID:  entity.PlayerID(req.PlayerID),
		Chips:     req.Chips,
	})

	if err != nil {
		logOperationCommandFailure(r, req.RequestID, "buy_in", req.SessionID, err)
		writeError(w, r, err)
		return
	}

	logOperationAcknowledgement(r, ack)
	writeJSON(w, http.StatusOK, ack)
}

func logOperationCommandFailure(r *http.Request, requestID, commandKind, sessionID string, err error) {
	slog.WarnContext(r.Context(), "operation_acknowledgement_failed",
		"request_id", requestID, "command_kind", commandKind,
		"session_id", sessionID, "err", err)
}

func logOperationAcknowledgement(r *http.Request, ack OperationAcknowledgement) {
	slog.InfoContext(r.Context(), "operation_acknowledged",
		"request_id", ack.RequestID, "command_kind", ack.Type,
		"session_id", ack.SessionID, "operation_id", ack.OperationID,
		"idempotent_replay", ack.IdempotentReplay)
}
