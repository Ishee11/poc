package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *Handler) GetSessionResults(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	res, err := h.getSessionResultsUC.Execute(usecase.GetSessionResultsQuery{
		SessionID: entity.SessionID(sessionID),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}
