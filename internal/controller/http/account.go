package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// Account godoc
// @Summary Current account
// @Description Returns the authenticated user and linked players.
// @Tags account
// @Produce json
// @Success 200 {object} AccountResponse
// @Failure 401 {object} ErrorResponse
// @Router /account [get]
func (h *AccountHandler) Account(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}

	principal, err := h.currentPrincipal(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	players, err := h.userPlayerLinksUC.ListUserPlayers(r.Context(), principal.UserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, AccountResponse{
		User:    authUserResponse(*principal),
		Players: playerResponses(players),
	})
}

// Players godoc
// @Summary Linked account players
// @Description Links, unlinks, or lists players linked to the current user.
// @Tags account
// @Accept json
// @Produce json
// @Success 200 {object} AccountPlayersResponse
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /account/players [get]
// @Router /account/players [post]
// @Router /account/players [delete]
func (h *AccountHandler) Players(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListPlayers(w, r)
	case http.MethodPost:
		h.LinkPlayer(w, r)
	case http.MethodDelete:
		h.UnlinkPlayer(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
	}
}

func (h *AccountHandler) ListPlayers(w http.ResponseWriter, r *http.Request) {
	principal, err := h.currentPrincipal(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	players, err := h.userPlayerLinksUC.ListUserPlayers(r.Context(), principal.UserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, AccountPlayersResponse{
		Players: playerResponses(players),
	})
}

func (h *AccountHandler) LinkPlayer(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	principal, err := h.currentPrincipal(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var req LinkAccountPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}
	if req.PlayerID == "" {
		writeErr(w, r, http.StatusBadRequest, "invalid_player_id", nil)
		return
	}

	if err := h.userPlayerLinksUC.LinkPlayer(r.Context(), usecase.LinkUserPlayerCommand{
		UserID:   principal.UserID,
		PlayerID: entity.PlayerID(req.PlayerID),
	}); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AccountHandler) UnlinkPlayer(w http.ResponseWriter, r *http.Request) {
	principal, err := h.currentPrincipal(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	playerID := r.URL.Query().Get("player_id")
	if playerID == "" {
		writeErr(w, r, http.StatusBadRequest, "invalid_player_id", nil)
		return
	}

	if err := h.userPlayerLinksUC.UnlinkPlayer(r.Context(), usecase.LinkUserPlayerCommand{
		UserID:   principal.UserID,
		PlayerID: entity.PlayerID(playerID),
	}); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AvailablePlayers godoc
// @Summary Unlinked players
// @Description Returns players that are not linked to any user.
// @Tags account
// @Produce json
// @Success 200 {object} AccountPlayersResponse
// @Failure 401 {object} ErrorResponse
// @Router /account/players/available [get]
func (h *AccountHandler) AvailablePlayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}

	if _, err := h.currentPrincipal(r); err != nil {
		writeError(w, r, err)
		return
	}

	h.writeAvailablePlayers(w, r)
}

// PublicAvailablePlayers godoc
// @Summary Unlinked players
// @Description Returns players that are not linked to any user.
// @Tags players
// @Produce json
// @Success 200 {object} AccountPlayersResponse
// @Router /players/unlinked [get]
func (h *AccountHandler) PublicAvailablePlayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}

	h.writeAvailablePlayers(w, r)
}

func (h *AccountHandler) writeAvailablePlayers(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		offset = 0
	}

	players, err := h.userPlayerLinksUC.ListUnlinkedPlayers(r.Context(), usecase.ListUnlinkedPlayersQuery{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, AccountPlayersResponse{
		Players: playerResponses(players),
	})
}

func (h *AccountHandler) currentPrincipal(r *http.Request) (*usecase.AuthPrincipal, error) {
	cookie, err := r.Cookie(h.cookie.Name)
	if err != nil {
		return nil, entity.ErrUnauthorized
	}

	return h.authUC.CurrentUser(r.Context(), cookie.Value)
}

func playerResponses(players []usecase.PlayerDTO) []PlayerDTO {
	result := make([]PlayerDTO, 0, len(players))
	for _, player := range players {
		result = append(result, PlayerDTO{
			ID:   player.ID,
			Name: player.Name,
		})
	}
	return result
}
