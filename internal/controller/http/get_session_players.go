package http

import (
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// GetSessionPlayers godoc
// @Summary Get session players
// @Description Returns players in a session
// @Tags sessions
// @Accept json
// @Produce json
// @Param session_id query string true "Session ID"
// @Success 200 {array} usecase.PlayerDTO
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /sessions/players [get]
func (h *SessionHandler) GetSessionPlayers(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	players, err := h.getSessionPlayersUC.Execute(usecase.GetSessionPlayersQuery{
		SessionID: entity.SessionID(sessionID),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, players)
}
