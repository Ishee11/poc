package http

import (
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	res, err := h.getSessionUC.Execute(usecase.GetSessionQuery{
		SessionID: entity.SessionID(sessionID),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	// 👇 ВАЖНО: используем тот же метод, что и в других handler'ах
	writeJSON(w, http.StatusOK, res)
}
