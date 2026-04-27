package http

import (
	"encoding/json"
	"net/http"

	"github.com/ishee11/poc/internal/usecase"
)

func (h *BlindClockHandler) GetActive(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetActive()
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *BlindClockHandler) Start(w http.ResponseWriter, r *http.Request) {
	h.writeMutation(w, r, h.service.Start)
}

func (h *BlindClockHandler) Pause(w http.ResponseWriter, r *http.Request) {
	h.writeMutation(w, r, h.service.Pause)
}

func (h *BlindClockHandler) Resume(w http.ResponseWriter, r *http.Request) {
	h.writeMutation(w, r, h.service.Resume)
}

func (h *BlindClockHandler) Reset(w http.ResponseWriter, r *http.Request) {
	h.writeMutation(w, r, h.service.Reset)
}

func (h *BlindClockHandler) PreviousLevel(w http.ResponseWriter, r *http.Request) {
	h.writeMutation(w, r, h.service.PreviousLevel)
}

func (h *BlindClockHandler) NextLevel(w http.ResponseWriter, r *http.Request) {
	h.writeMutation(w, r, h.service.NextLevel)
}

func (h *BlindClockHandler) UpdateLevels(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req UpdateBlindClockLevelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	input := make([]usecase.BlindClockLevelInput, 0, len(req.Levels))
	for _, level := range req.Levels {
		input = append(input, usecase.BlindClockLevelInput{
			SmallBlind:      level.SmallBlind,
			BigBlind:        level.BigBlind,
			DurationMinutes: level.DurationMinutes,
		})
	}

	resp, err := h.service.UpdateLevels(input)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *BlindClockHandler) writeMutation(
	w http.ResponseWriter,
	r *http.Request,
	fn func() (*usecase.BlindClockResponse, error),
) {
	resp, err := fn()
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
