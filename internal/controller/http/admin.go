package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// Accounts godoc
// @Summary List account ownership
// @Tags admin
// @Produce json
// @Param query query string false "Email or player search"
// @Param limit query int false "Page size"
// @Param offset query int false "Page offset"
// @Success 200 {object} usecase.AdminAccountList
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /admin/accounts [get]
func (h *AdminHandler) Accounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if h.requireAdminPrincipal(w, r) == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	result, err := h.accountOwnershipUC.ListAccounts(r.Context(), usecase.AdminListAccountsQuery{
		Query: r.URL.Query().Get("query"), Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AccountPlayer godoc
// @Summary Replace or clear account player ownership
// @Tags admin
// @Accept json
// @Produce json
// @Param user_id path string true "Account ID"
// @Param request body AdminReplaceAccountPlayerRequest true "Player assignment"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /admin/accounts/{user_id}/player [put]
// @Router /admin/accounts/{user_id}/player [delete]
func (h *AdminHandler) AccountPlayer(w http.ResponseWriter, r *http.Request) {
	userID, ok := adminAccountPlayerPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	principal := h.requireAdminPrincipal(w, r)
	if principal == nil {
		return
	}

	var change usecase.OwnershipChange
	var err error
	operation := "admin_clear"
	switch r.Method {
	case http.MethodPut:
		defer r.Body.Close()
		var req AdminReplaceAccountPlayerRequest
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
			return
		}
		operation = "admin_replace"
		change, err = h.accountOwnershipUC.Replace(r.Context(), userID, entity.PlayerID(req.PlayerID))
	case http.MethodDelete:
		change, err = h.accountOwnershipUC.Clear(r.Context(), userID)
	default:
		w.Header().Set("Allow", "PUT, DELETE")
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if err != nil {
		writeError(w, r, err)
		return
	}

	slog.InfoContext(
		r.Context(),
		"account_player_ownership_changed",
		"request_id", GetRequestID(r.Context()),
		"operation", operation,
		"actor_user_id", principal.UserID,
		"target_user_id", change.TargetUserID,
		"old_player_id", ownershipPlayerID(change.OldPlayerID),
		"new_player_id", ownershipPlayerID(change.NewPlayerID),
	)
	w.WriteHeader(http.StatusNoContent)
}

func adminAccountPlayerPath(path string) (entity.AuthUserID, bool) {
	raw := strings.TrimPrefix(path, "/admin/accounts/")
	parts := strings.Split(raw, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "player" {
		return "", false
	}
	decoded, err := url.PathUnescape(parts[0])
	if err != nil || decoded == "" {
		return "", false
	}
	return entity.AuthUserID(decoded), true
}

func ownershipPlayerID(playerID *entity.PlayerID) string {
	if playerID == nil {
		return ""
	}
	return string(*playerID)
}

func (h *AdminHandler) RenamePlayer(w http.ResponseWriter, r *http.Request) {
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

	if err := h.renamePlayerUC.Execute(r.Context(), entity.PlayerID(playerID), req.Name); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) UpdateSessionConfig(w http.ResponseWriter, r *http.Request) {
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

	if err := h.updateSessionConfigUC.Execute(r.Context(), entity.SessionID(sessionID), req.ChipRate, req.BigBlind, entity.Currency(req.Currency)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) DeletePlayer(w http.ResponseWriter, r *http.Request) {
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

	if err := h.deletePlayerUC.Execute(r.Context(), entity.PlayerID(playerID)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
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

	if err := h.deleteSessionUC.Execute(r.Context(), entity.SessionID(sessionID)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) DeleteSessionFinish(w http.ResponseWriter, r *http.Request) {
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

	if err := h.deleteSessionFinishUC.Execute(r.Context(), entity.SessionID(sessionID)); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	return h.requireAdminPrincipal(w, r) != nil
}

func (h *AdminHandler) requireAdminPrincipal(w http.ResponseWriter, r *http.Request) *usecase.AuthPrincipal {
	cookie, err := r.Cookie(h.cookie.Name)
	if err != nil || cookie.Value == "" {
		writeError(w, r, entity.ErrUnauthorized)
		return nil
	}

	principal, err := h.authUC.CurrentUser(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, r, err)
		return nil
	}

	if err := h.authUC.RequireRole(*principal, entity.AuthRoleAdmin); err != nil {
		writeError(w, r, err)
		return nil
	}

	return principal
}
