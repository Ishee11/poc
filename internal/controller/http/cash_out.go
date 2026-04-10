package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *Handler) CashOut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestID string `json:"request_id"`
		SessionID string `json:"session_id"`
		PlayerID  string `json:"player_id"`
		Chips     int64  `json:"chips"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err := h.cashOutUC.Execute(usecase.CashOutCommand{
		RequestID: req.RequestID,
		SessionID: entity.SessionID(req.SessionID),
		PlayerID:  entity.PlayerID(req.PlayerID),
		Chips:     req.Chips,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
