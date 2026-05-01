package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/usecase/command"
)

// CreatePlayer godoc
// @Summary Create player
// @Description Creates a new player
// @Tags players
// @Accept json
// @Produce json
// @Param request body CreatePlayerRequest true "Create player request"
// @Success 200 {object} CreatePlayerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /players [post]
func (h *PlayerHandler) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req CreatePlayerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	// базовая валидация
	if req.RequestID == "" || req.Name == "" {
		writeErr(w, r, http.StatusBadRequest, "invalid_request", nil)
		return
	}

	id, err := h.createPlayerUC.Execute(r.Context(), command.CreatePlayerCommand{
		RequestID: req.RequestID,
		Name:      req.Name,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, CreatePlayerResponse{
		PlayerID: id,
	})
}
