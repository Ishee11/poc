package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
)

func writeError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	var balancedErr *entity.SessionNotBalancedError
	if errors.As(err, &balancedErr) {
		writeErr(w, http.StatusConflict, "session_not_balanced", map[string]any{
			"remaining_chips": balancedErr.RemainingChips,
		})
		return
	}

	switch {
	case errors.Is(err, entity.ErrSessionNotFound):
		writeErr(w, http.StatusNotFound, "session_not_found", nil)

	case errors.Is(err, entity.ErrSessionAlreadyExists):
		writeErr(w, http.StatusConflict, "session_already_exists", nil)

	case errors.Is(err, entity.ErrSessionFinished):
		writeErr(w, http.StatusConflict, "session_finished", nil)

	case errors.Is(err, entity.ErrSessionNotActive):
		writeErr(w, http.StatusConflict, "session_not_active", nil)

	case errors.Is(err, entity.ErrInvalidRequestID):
		writeErr(w, http.StatusBadRequest, "invalid_request_id", nil)

	case errors.Is(err, entity.ErrInvalidChips):
		writeErr(w, http.StatusBadRequest, "invalid_chips", nil)

	case errors.Is(err, entity.ErrPlayerNotFound):
		writeErr(w, http.StatusNotFound, "player_not_found", nil)

	case errors.Is(err, entity.ErrInvalidCashOut):
		writeErr(w, http.StatusBadRequest, "invalid_cash_out", nil)

	case errors.Is(err, entity.ErrInvalidOperation):
		writeErr(w, http.StatusBadRequest, "invalid_operation", nil)

	case errors.Is(err, entity.ErrOperationNotFound):
		writeErr(w, http.StatusNotFound, "operation_not_found", nil)

	case errors.Is(err, entity.ErrOperationAlreadyReversed):
		writeErr(w, http.StatusConflict, "operation_already_reversed", nil)

	case errors.Is(err, entity.ErrDuplicateRequest):
		w.WriteHeader(http.StatusOK)

	default:
		writeErr(w, http.StatusInternalServerError, "internal_error", nil)
	}
}

// --- helper ---
func writeErr(w http.ResponseWriter, status int, code string, details any) {
	writeJSON(w, status, ErrorResponse{
		Error:   code,
		Details: details,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
