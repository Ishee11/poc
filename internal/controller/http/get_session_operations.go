package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *Handler) GetSessionOperations(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	res, err := h.getSessionOpsUC.Execute(usecase.GetSessionOperationsQuery{
		SessionID: entity.SessionID(sessionID),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	json.NewEncoder(w).Encode(res)
}
