package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

func (h *Handler) BuyIn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestID  string `json:"request_id"`
		SessionID  string `json:"session_id"`
		PlayerName string `json:"player_name"`
		Chips      int64  `json:"chips"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err := h.buyInUC.Execute(command.BuyInCommand{
		RequestID:  req.RequestID,
		SessionID:  entity.SessionID(req.SessionID),
		PlayerName: req.PlayerName,
		Chips:      req.Chips,
	})

	if err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
