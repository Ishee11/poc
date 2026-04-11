package http

import (
	"io/fs"
	"net/http"

	"github.com/ishee11/poc/web"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	sub, _ := fs.Sub(web.FS, ".")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, sub, "index.html")
	})

	// static (если будут css/js)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web"))))

	mux.HandleFunc("/health", h.Health)

	mux.HandleFunc("/session/start", h.StartSession)
	mux.HandleFunc("/session/finish", h.FinishSession)

	mux.HandleFunc("/buy-in", h.BuyIn)
	mux.HandleFunc("/cash-out", h.CashOut)
	mux.HandleFunc("/operation/reverse", h.ReverseOperation)

	mux.HandleFunc("/session", h.GetSession)
	mux.HandleFunc("/session/operations", h.GetSessionOperations)
	mux.HandleFunc("/session/results", h.GetSessionResults)
	mux.HandleFunc("/stats/sessions", h.GetStatsSessions)
	mux.HandleFunc("/stats/players", h.GetStatsPlayers)
	mux.HandleFunc("/stats/player", h.GetPlayerStats)

	var handler http.Handler = mux
	handler = CORSMiddleware(handler)
	handler = RequestIDMiddleware(handler)
	handler = LoggingMiddleware(handler)

	return handler
}
