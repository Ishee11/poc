package http

import (
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *Handler) GetSessionPlayers(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	players, err := h.getSessionPlayersUC.Execute(usecase.GetSessionPlayersQuery{
		SessionID: entity.SessionID(sessionID),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, players)
}
