package http

import (
	"io/fs"
	"net/http"

	"github.com/ishee11/poc/web"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// ===== STATIC =====
	sub, err := fs.Sub(web.FS, ".")
	if err != nil {
		panic(err)
	}

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))

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

	// ===== SESSIONS =====
	mux.HandleFunc("/sessions/start", h.Session.StartSession)
	mux.HandleFunc("/sessions/finish", h.Session.FinishSession)
	mux.HandleFunc("/sessions", h.Session.GetSession)
	mux.HandleFunc("/sessions/operations", h.Session.GetSessionOperations)
	mux.HandleFunc("/sessions/players", h.Session.GetSessionPlayers)

	// ===== OPERATIONS =====
	mux.HandleFunc("/operations/buy-in", h.Operation.BuyIn)
	mux.HandleFunc("/operations/cash-out", h.Operation.CashOut)
	mux.HandleFunc("/operations/reverse", h.Operation.ReverseOperation)

	// ===== PLAYERS =====
	mux.HandleFunc("/players", h.Player.CreatePlayer)
	mux.HandleFunc("/players/stats", h.Player.GetPlayerStats)

	// ===== STATS =====
	mux.HandleFunc("/stats/sessions", h.Stats.GetStatsSessions)
	mux.HandleFunc("/stats/players", h.Stats.GetStatsPlayers)

	// ===== MIDDLEWARE =====
	var handler http.Handler = mux
	handler = CORSMiddleware(handler)
	handler = RequestIDMiddleware(handler)
	handler = LoggingMiddleware(handler)

	return handler
}
