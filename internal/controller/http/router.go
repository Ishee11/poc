package http

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)

	mux.HandleFunc("/session/start", h.StartSession)
	mux.HandleFunc("/session/finish", h.FinishSession)

	mux.HandleFunc("/buy-in", h.BuyIn)
	mux.HandleFunc("/cash-out", h.CashOut)
	mux.HandleFunc("/operation/reverse", h.ReverseOperation)

	mux.HandleFunc("/session", h.GetSession)
	mux.HandleFunc("/session/operations", h.GetSessionOperations)
	mux.HandleFunc("/session/results", h.GetSessionResults)

	// middleware chain
	var handler http.Handler = mux
	handler = RequestIDMiddleware(handler)
	handler = LoggingMiddleware(handler)

	return handler
}
