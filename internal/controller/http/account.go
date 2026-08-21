package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// Account godoc
// @Summary Current account
// @Description Returns the authenticated user, nullable singular player ownership, onboarding state, and transitional zero-or-one players mirror.
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
	identities, err := h.telegramAuthUC.ListIdentities(r.Context(), principal.UserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	playerList := playerResponses(players)
	var player *PlayerDTO
	if len(playerList) > 0 {
		player = &playerList[0]
	}
	writeJSON(w, http.StatusOK, AccountResponse{
		User:               authUserResponse(*principal),
		Player:             player,
		OnboardingRequired: player == nil,
		Players:            playerList,
		Identities:         identities,
	})
}

func (h *AccountHandler) TelegramIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	principal, err := h.currentPrincipal(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.telegramAuthUC.Unlink(r.Context(), principal.UserID); err != nil {
		writeError(w, r, err)
		return
	}
	slog.InfoContext(r.Context(), "telegram_identity_unlinked", "request_id", GetRequestID(r.Context()), "user_id", principal.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// Player godoc
// @Summary Establish current account player ownership
// @Description Claims an unowned existing player or creates a new owned player. Ownership is write-once.
// @Tags account
// @Accept json
// @Produce json
// @Param request body SelectAccountPlayerRequest true "Player selection"
// @Success 200 {object} AccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /account/player [put]
func (h *AccountHandler) Player(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	h.choosePlayer(w, r, false)
}

// Players godoc
// @Summary Transitional account player compatibility
// @Description Lists the zero-or-one owned player or applies the legacy one-time existing-player claim alias. Self-service deletion is disabled.
// @Tags account
// @Accept json
// @Produce json
// @Success 200 {object} AccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /account/players [get]
// @Router /account/players [post]
func (h *AccountHandler) Players(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListPlayers(w, r)
	case http.MethodPost:
		h.choosePlayer(w, r, true)
	case http.MethodDelete:
		w.Header().Set("Allow", "GET, POST")
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
	default:
		w.Header().Set("Allow", "GET, POST")
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

func (h *AccountHandler) choosePlayer(w http.ResponseWriter, r *http.Request, legacy bool) {
	defer r.Body.Close()

	principal, err := h.currentPrincipal(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var selection usecase.PlayerSelection
	if legacy {
		var req LinkAccountPlayerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
			return
		}
		selection = usecase.PlayerSelection{
			Mode:     usecase.PlayerSelectionExisting,
			PlayerID: entity.PlayerID(req.PlayerID),
		}
	} else if err := json.NewDecoder(r.Body).Decode(&selection); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	player, err := h.userPlayerLinksUC.ChooseOrCreatePlayer(r.Context(), principal.UserID, selection)
	if err != nil {
		writeError(w, r, err)
		return
	}

	slog.InfoContext(
		r.Context(),
		"account_player_ownership_changed",
		"request_id", GetRequestID(r.Context()),
		"operation", "self_claim",
		"actor_user_id", principal.UserID,
		"target_user_id", principal.UserID,
		"old_player_id", "",
		"new_player_id", player.ID,
	)
	playerResponse := PlayerDTO{ID: player.ID, Name: player.Name}
	writeJSON(w, http.StatusOK, AccountResponse{
		User:               authUserResponse(*principal),
		Player:             &playerResponse,
		OnboardingRequired: false,
		Players:            []PlayerDTO{playerResponse},
	})
}

// AvailablePlayers godoc
// @Summary Unlinked players
// @Description Returns players that are not linked to any user.
// @Tags account
// @Produce json
// @Success 200 {object} AvailablePlayersResponse
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
// @Success 200 {object} AvailablePlayersResponse
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

	writeJSON(w, http.StatusOK, AvailablePlayersResponse{
		Players: availablePlayerResponses(players),
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

func availablePlayerResponses(players []usecase.AvailablePlayerDTO) []AvailablePlayerDTO {
	result := make([]AvailablePlayerDTO, 0, len(players))
	for _, player := range players {
		result = append(result, AvailablePlayerDTO{
			ID:            player.ID,
			Name:          player.Name,
			SessionsCount: player.SessionsCount,
			LastPlayedAt:  player.LastPlayedAt,
		})
	}
	return result
}
