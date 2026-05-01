package http

import (
	"encoding/json"
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
// @Success 204 "No Content"
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

	err := h.buyInUC.Execute(r.Context(), command.BuyInCommand{
		RequestID: req.RequestID,
		SessionID: entity.SessionID(req.SessionID),
		PlayerID:  entity.PlayerID(req.PlayerID),
		Chips:     req.Chips,
	})

	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
