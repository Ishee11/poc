package http

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/buy-in", h.BuyIn)
	mux.HandleFunc("/start-session", h.StartSession)
	mux.HandleFunc("/session", h.GetSession)

	return mux
}
