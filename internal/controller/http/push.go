package http

import (
	"encoding/json"
	"net/http"
)

func (h *PushHandler) Config(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.GetClientConfig())
}

func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req PushSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	if err := h.service.Subscribe(req.toInput()); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req PushUnsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	if err := h.service.Unsubscribe(req.Endpoint); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
