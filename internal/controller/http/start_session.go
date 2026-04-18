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
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}

	err := h.startSessionUC.Execute(command.StartSessionCommand{
		SessionID: entity.SessionID(req.SessionID),
		ChipRate:  req.ChipRate,
	})

	if err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
