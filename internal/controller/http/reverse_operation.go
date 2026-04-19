package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

// ReverseOperation godoc
// @Summary Reverse operation
// @Description Reverses a target operation
// @Tags operations
// @Accept json
// @Produce json
// @Param request body ReverseOperationRequest true "Reverse request"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /operations/reverse [post]
func (h *OperationHandler) ReverseOperation(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Header.Get("Content-Type") != "application/json" {
		writeErr(w, r, http.StatusUnsupportedMediaType, "unsupported_content_type", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req ReverseOperationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	if req.RequestID == "" || req.TargetOperationID == "" {
		writeErr(w, r, http.StatusBadRequest, "invalid_request", nil)
		return
	}

	err := h.reverseOperationUC.Execute(command.ReverseOperationCommand{
		RequestID:         req.RequestID,
		TargetOperationID: entity.OperationID(req.TargetOperationID),
	})

	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
