package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *Handler) ReverseOperation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestID         string `json:"request_id"`
		OperationID       string `json:"operation_id"`
		TargetOperationID string `json:"target_operation_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err := h.reverseOperationUC.Execute(usecase.ReverseOperationCommand{
		RequestID:         req.RequestID,
		OperationID:       entity.OperationID(req.OperationID),
		TargetOperationID: entity.OperationID(req.TargetOperationID),
	})

	if err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
