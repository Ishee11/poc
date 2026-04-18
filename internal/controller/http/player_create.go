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
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// базовая валидация
	if req.RequestID == "" || req.Name == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	id, err := h.createPlayerUC.Execute(command.CreatePlayerCommand{
		RequestID: req.RequestID,
		Name:      req.Name,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, CreatePlayerResponse{
		PlayerID: id,
	})
}
