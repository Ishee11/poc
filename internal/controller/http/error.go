package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
)

type apiError struct {
	status  int
	code    string
	details any
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	apiErr := mapError(err)
	if apiErr.status >= http.StatusInternalServerError {
		slog.ErrorContext(
			r.Context(),
			"request_error",
			"request_id", GetRequestID(r.Context()),
			"error_code", apiErr.code,
			"err", err,
		)
	}

	writeErr(w, r, apiErr.status, apiErr.code, apiErr.details)
}

func mapError(err error) apiError {
	var balancedErr *entity.SessionNotBalancedError
	if errors.As(err, &balancedErr) {
		return apiError{
			status: http.StatusConflict,
			code:   "session_not_balanced",
			details: map[string]any{
				"remaining_chips": balancedErr.RemainingChips,
			},
		}
	}

	switch {
	case errors.Is(err, entity.ErrSessionNotFound):
		return apiError{status: http.StatusNotFound, code: "session_not_found"}

	case errors.Is(err, entity.ErrSessionAlreadyExists):
		return apiError{status: http.StatusConflict, code: "session_already_exists"}

	case errors.Is(err, entity.ErrSessionFinished):
		return apiError{status: http.StatusConflict, code: "session_finished"}

	case errors.Is(err, entity.ErrSessionNotActive):
		return apiError{status: http.StatusConflict, code: "session_not_active"}

	case errors.Is(err, entity.ErrSessionNotCreated):
		return apiError{status: http.StatusConflict, code: "session_not_created"}

	case errors.Is(err, entity.ErrSessionNotFinished):
		return apiError{status: http.StatusConflict, code: "session_not_finished"}

	case errors.Is(err, entity.ErrInvalidRequestID):
		return apiError{status: http.StatusBadRequest, code: "invalid_request_id"}

	case errors.Is(err, entity.ErrInvalidChips):
		return apiError{status: http.StatusBadRequest, code: "invalid_chips"}

	case errors.Is(err, entity.ErrInvalidChipAmount):
		return apiError{status: http.StatusBadRequest, code: "invalid_chip_amount"}

	case errors.Is(err, entity.ErrPlayerNotFound):
		return apiError{status: http.StatusNotFound, code: "player_not_found"}

	case errors.Is(err, entity.ErrPlayerAlreadyLinked):
		return apiError{status: http.StatusConflict, code: "player_already_linked"}

	case errors.Is(err, entity.ErrUserPlayerNotLinked):
		return apiError{status: http.StatusNotFound, code: "user_player_not_linked"}

	case errors.Is(err, entity.ErrPlayerNotInGame):
		return apiError{status: http.StatusConflict, code: "player_not_in_game"}

	case errors.Is(err, entity.ErrPlayerStillInGame):
		return apiError{status: http.StatusConflict, code: "player_still_in_game"}

	case errors.Is(err, entity.ErrPlayersStillInGame):
		return apiError{status: http.StatusConflict, code: "players_still_in_game"}

	case errors.Is(err, entity.ErrInvalidPlayerID):
		return apiError{status: http.StatusBadRequest, code: "invalid_player_id"}

	case errors.Is(err, entity.ErrInvalidPlayerName):
		return apiError{status: http.StatusBadRequest, code: "invalid_player_name"}

	case errors.Is(err, entity.ErrInvalidCashOut):
		return apiError{status: http.StatusBadRequest, code: "invalid_cash_out"}

	case errors.Is(err, entity.ErrInvalidOperation):
		return apiError{status: http.StatusBadRequest, code: "invalid_operation"}

	case errors.Is(err, entity.ErrInvalidOperationType):
		return apiError{status: http.StatusBadRequest, code: "invalid_operation_type"}

	case errors.Is(err, entity.ErrInvalidReference):
		return apiError{status: http.StatusBadRequest, code: "invalid_reference"}

	case errors.Is(err, entity.ErrOperationNotFound):
		return apiError{status: http.StatusNotFound, code: "operation_not_found"}

	case errors.Is(err, entity.ErrOperationAlreadyReversed):
		return apiError{status: http.StatusConflict, code: "operation_already_reversed"}

	case errors.Is(err, entity.ErrNotEnoughChipsOnTable):
		return apiError{status: http.StatusConflict, code: "not_enough_chips_on_table"}

	case errors.Is(err, entity.ErrInsufficientTableChips):
		return apiError{status: http.StatusConflict, code: "insufficient_table_chips"}

	case errors.Is(err, entity.ErrInvalidMoney):
		return apiError{status: http.StatusBadRequest, code: "invalid_money"}

	case errors.Is(err, entity.ErrBlindClockNotFound):
		return apiError{status: http.StatusNotFound, code: "blind_clock_not_found"}

	case errors.Is(err, entity.ErrBlindClockHasNoLevels):
		return apiError{status: http.StatusBadRequest, code: "blind_clock_has_no_levels"}

	case errors.Is(err, entity.ErrBlindClockAlreadyRunning):
		return apiError{status: http.StatusConflict, code: "blind_clock_already_running"}

	case errors.Is(err, entity.ErrBlindClockNotRunning):
		return apiError{status: http.StatusConflict, code: "blind_clock_not_running"}

	case errors.Is(err, entity.ErrBlindClockNotPaused):
		return apiError{status: http.StatusConflict, code: "blind_clock_not_paused"}

	case errors.Is(err, entity.ErrBlindClockFinished):
		return apiError{status: http.StatusConflict, code: "blind_clock_finished"}

	case errors.Is(err, entity.ErrInvalidBlindClockLevel):
		return apiError{status: http.StatusBadRequest, code: "invalid_blind_clock_level"}

	case errors.Is(err, entity.ErrBlindClockLevelsLocked):
		return apiError{status: http.StatusConflict, code: "blind_clock_levels_locked"}

	case errors.Is(err, entity.ErrUnbalancedSession):
		return apiError{status: http.StatusConflict, code: "unbalanced_session"}

	case errors.Is(err, entity.ErrTableNotSettled):
		return apiError{status: http.StatusConflict, code: "table_not_settled"}

	case errors.Is(err, entity.ErrDuplicateRequest):
		return apiError{status: http.StatusOK, code: "duplicate_request"}

	case errors.Is(err, entity.ErrInvalidCredentials):
		return apiError{status: http.StatusUnauthorized, code: "invalid_credentials"}

	case errors.Is(err, entity.ErrAuthUserAlreadyExists):
		return apiError{status: http.StatusConflict, code: "user_already_exists"}

	case errors.Is(err, entity.ErrInvalidAuthEmail):
		return apiError{status: http.StatusBadRequest, code: "invalid_auth_email"}

	case errors.Is(err, entity.ErrPasswordTooShort):
		return apiError{status: http.StatusBadRequest, code: "password_too_short"}

	case errors.Is(err, entity.ErrUnauthorized):
		return apiError{status: http.StatusUnauthorized, code: "unauthorized"}

	case errors.Is(err, entity.ErrForbidden):
		return apiError{status: http.StatusForbidden, code: "forbidden"}

	case errors.Is(err, entity.ErrAuthRateLimited):
		return apiError{status: http.StatusTooManyRequests, code: "rate_limited"}

	default:
		return apiError{status: http.StatusInternalServerError, code: "internal_error"}
	}
}

// --- helper ---
func writeErr(w http.ResponseWriter, r *http.Request, status int, code string, details any) {
	setResponseErrorCode(w, code)
	if status == http.StatusOK && details == nil {
		w.WriteHeader(status)
		return
	}

	writeJSON(w, status, ErrorResponse{
		Error:     code,
		RequestID: GetRequestID(r.Context()),
		Details:   details,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
