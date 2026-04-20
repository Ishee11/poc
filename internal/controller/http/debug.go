package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
)

func (h *DebugHandler) RenamePlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}

	playerID := r.URL.Query().Get("player_id")
	if playerID == "" {
		writeErr(w, r, http.StatusBadRequest, "player_id_required", nil)
		return
	}

	var req RenamePlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	if err := h.renamePlayerUC.Execute(entity.PlayerID(playerID), req.Name); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DebugHandler) UpdateSessionConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}

	var req UpdateSessionConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	if err := h.updateSessionConfigUC.Execute(entity.SessionID(sessionID), req.ChipRate, req.BigBlind, entity.Currency(req.Currency)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DebugHandler) DeletePlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}

	playerID := r.URL.Query().Get("player_id")
	if playerID == "" {
		writeErr(w, r, http.StatusBadRequest, "player_id_required", nil)
		return
	}

	if err := h.deletePlayerUC.Execute(entity.PlayerID(playerID)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DebugHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}

	if err := h.deleteSessionUC.Execute(entity.SessionID(sessionID)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DebugHandler) DeleteSessionFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}

	if err := h.deleteSessionFinishUC.Execute(entity.SessionID(sessionID)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DebugHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie(h.cookie.Name)
	if err != nil || cookie.Value == "" {
		writeError(w, r, entity.ErrUnauthorized)
		return false
	}

	principal, err := h.authUC.CurrentUser(cookie.Value)
	if err != nil {
		writeError(w, r, err)
		return false
	}

	if err := h.authUC.RequireRole(*principal, entity.AuthRoleAdmin); err != nil {
		writeError(w, r, err)
		return false
	}

	return true
}
