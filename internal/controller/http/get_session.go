package http

import (
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// GetSession godoc
// @Summary Get session
// @Description Get session by ID. Finished sessions are public; active sessions require participant or selected guest-player context. can_mutate is scoped to the current authenticated viewer.
// @Tags sessions
// @Accept json
// @Produce json
// @Param session_id query string true "Session ID"
// @Success 200 {object} usecase.GetSessionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /sessions [get]
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	if sessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}
	if !h.access.requireView(w, r, entity.SessionID(sessionID)) {
		return
	}

	res, err := h.getSessionUC.Execute(r.Context(), usecase.GetSessionQuery{
		SessionID: entity.SessionID(sessionID),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	res.CanMutate, err = h.access.canMutate(r, entity.SessionID(sessionID))
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}
