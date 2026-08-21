package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *OperationHandler) SettlementTransfers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListSettlementTransfers(w, r)
	case http.MethodPut:
		h.SaveSettlementTransfers(w, r)
	default:
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
	}
}

func (h *OperationHandler) ListSettlementTransfers(w http.ResponseWriter, r *http.Request) {
	sessionID := entity.SessionID(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}
	if !h.access.requireView(w, r, sessionID) {
		return
	}
	transfers, err := h.settlementTransferService.List(r.Context(), sessionID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, transfers)
}

func (h *OperationHandler) SaveSettlementTransfers(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req SaveSettlementTransfersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}
	if req.SessionID == "" {
		writeErr(w, r, http.StatusBadRequest, "session_id_required", nil)
		return
	}
	if !h.access.requireView(w, r, entity.SessionID(req.SessionID)) {
		return
	}

	transfers := make([]usecase.SettlementTransfer, 0, len(req.Transfers))
	for _, transfer := range req.Transfers {
		transfers = append(transfers, usecase.SettlementTransfer{
			ID:     transfer.ID,
			From:   entity.PlayerID(transfer.From),
			To:     entity.PlayerID(transfer.To),
			Amount: transfer.Amount,
		})
	}

	saved, err := h.settlementTransferService.Save(r.Context(), usecase.SaveSettlementTransfersInput{
		SessionID: entity.SessionID(req.SessionID),
		Transfers: transfers,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, saved)
}
