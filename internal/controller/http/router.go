package http

import (
	"io/fs"
	"net/http"

	"github.com/ishee11/poc/web"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// embed FS
	sub, err := fs.Sub(web.FS, ".")
	if err != nil {
		panic(err)
	}

	// ===== STATIC (CSS / JS) =====
	//mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))
	fs := http.FS(web.FS)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(fs)))

	// ===== INDEX =====
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, sub, "index.html")
	})

	// ===== HEALTH =====
	mux.HandleFunc("/health", h.Health)

	// ===== WRITE =====
	mux.HandleFunc("/session/start", h.StartSession)
	mux.HandleFunc("/session/finish", h.FinishSession)

	mux.HandleFunc("/buy-in", h.BuyIn)
	mux.HandleFunc("/cash-out", h.CashOut)
	mux.HandleFunc("/operation/reverse", h.ReverseOperation)

	// ===== READ =====
	mux.HandleFunc("/session", h.GetSession)
	mux.HandleFunc("/session/operations", h.GetSessionOperations)
	mux.HandleFunc("/session/results", h.GetSessionResults)
	mux.HandleFunc("/stats/sessions", h.GetStatsSessions)
	mux.HandleFunc("/session/players", h.GetSessionPlayers)

	mux.HandleFunc("/stats/players", h.GetStatsPlayers)
	mux.HandleFunc("/stats/player", h.GetPlayerStats)

	// ===== MIDDLEWARE =====
	var handler http.Handler = mux
	handler = CORSMiddleware(handler)
	handler = RequestIDMiddleware(handler)
	handler = LoggingMiddleware(handler)

	return handler
}
