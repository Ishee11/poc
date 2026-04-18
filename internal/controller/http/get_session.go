package http

import (
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// GetSession godoc
// @Summary Get session
// @Description Get session by ID
// @Tags sessions
// @Accept json
// @Produce json
// @Param session_id query string true "Session ID"
// @Success 200 {object} usecase.GetSessionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /sessions [get]
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	res, err := h.getSessionUC.Execute(usecase.GetSessionQuery{
		SessionID: entity.SessionID(sessionID),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}
