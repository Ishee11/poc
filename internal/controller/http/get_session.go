package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	res, err := h.getSessionUC.Execute(usecase.GetSessionQuery{
		SessionID: entity.SessionID(sessionID),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"session_id":     res.SessionID,
		"status":         res.Status,
		"chip_rate":      res.ChipRate,
		"total_buy_in":   res.TotalBuyIn,
		"total_cash_out": res.TotalCashOut,
		"total_chips":    res.TotalChips,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
