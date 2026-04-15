package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
)

func writeError(w http.ResponseWriter, err error) {
	switch e := err.(type) {

	// --- кастомная бизнес-ошибка ---
	case *entity.SessionNotBalancedError:
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "session not balanced",
			"details": map[string]any{
				"remaining_chips": e.RemainingChips,
			},
		})
		return

	// --- обычные ошибки ---
	case nil:
		return
	}

	switch {
	case errors.Is(err, entity.ErrSessionNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": err.Error(),
		})

	case errors.Is(err, entity.ErrInvalidChips),
		errors.Is(err, entity.ErrInvalidCashOut),
		errors.Is(err, entity.ErrInvalidOperation),
		errors.Is(err, entity.ErrInvalidRequestID):
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": err.Error(),
		})

	case errors.Is(err, entity.ErrDuplicateRequest):
		w.WriteHeader(http.StatusOK)

	default:
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": err.Error(),
		})
	}
}

// --- helper ---
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
