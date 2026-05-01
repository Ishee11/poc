package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

// FinishSession godoc
// @Summary Finish session
// @Description Finish poker session (must be balanced)
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body FinishSessionRequest true "Finish session request"
// @Success 200
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/finish [post]
func (h *SessionHandler) FinishSession(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// защита от слишком большого тела (опционально, но правильно)
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB

	var req FinishSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	if req.RequestID == "" {
		writeErr(w, r, http.StatusBadRequest, "request_id_required", nil)
		return
	}

	if req.SessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}

	err := h.finishSessionUC.Execute(r.Context(), command.FinishSessionCommand{
		RequestID: req.RequestID,
		SessionID: entity.SessionID(req.SessionID),
	})

	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
