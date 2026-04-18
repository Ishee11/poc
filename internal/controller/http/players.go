package http

import (
	"net/http"
	"strconv"

	"github.com/ishee11/poc/internal/usecase"
)

func (h *PlayerHandler) Players(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetPlayers(w, r)
	case http.MethodPost:
		h.CreatePlayer(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *PlayerHandler) GetPlayers(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		offset = 0
	}

	res, err := h.getPlayersUC.Execute(usecase.GetPlayersQuery{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}
