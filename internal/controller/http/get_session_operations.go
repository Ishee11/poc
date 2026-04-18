package http

import (
	"net/http"
	"strconv"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// GetSessionOperations godoc
// @Summary Get session operations
// @Description List operations for session
// @Tags operations
// @Accept json
// @Produce json
// @Param session_id query string true "Session ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} usecase.OperationDTO
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /operations [get]
func (h *SessionHandler) GetSessionOperations(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		offset = 0
	}

	res, err := h.getSessionOpsUC.Execute(usecase.GetSessionOperationsQuery{
		SessionID: entity.SessionID(sessionID),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}
