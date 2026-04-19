package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

// StartSession godoc
// @Summary Start session
// @Description Create new poker session
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body StartSessionRequest true "start session request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/start [post]
func (h *SessionHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req StartSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	id, err := h.startSessionUC.Execute(command.StartSessionCommand{
		ChipRate: req.ChipRate,
		BigBlind: req.BigBlind,
		Currency: entity.Currency(req.Currency),
	})

	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": id,
	})
}
