package http

import (
	"io/fs"
	"net/http"

	"github.com/ishee11/poc/web"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/ishee11/poc/docs"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// ===== STATIC =====
	sub, err := fs.Sub(web.FS, ".")
	if err != nil {
		panic(err)
	}

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))
	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		http.ServeFileFS(w, r, sub, "sw.js")
	})
	mux.HandleFunc("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		http.ServeFileFS(w, r, sub, "manifest.webmanifest")
	})

	// ===== INDEX =====
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !isClientRoute(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, sub, "index.html")
	})

	// ===== HEALTH =====
	mux.HandleFunc("/health", h.Health)
	mux.Handle("/metrics", MetricsHandler())

	// ===== AUTH =====
	mux.HandleFunc("/auth/register", h.Auth.Register)
	mux.HandleFunc("/auth/login", h.Auth.Login)
	mux.HandleFunc("/auth/logout", h.Auth.Logout)
	mux.HandleFunc("/auth/me", h.Auth.Me)

	// ===== ACCOUNT =====
	mux.HandleFunc("/account", h.Account.Account)
	mux.HandleFunc("/account/players", h.Account.Players)
	mux.HandleFunc("/account/players/available", h.Account.AvailablePlayers)

	// ===== SWAGGER =====
	mux.HandleFunc("/swagger/", httpSwagger.Handler())

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

	// ===== BLINDS CLOCK =====
	mux.HandleFunc("/blinds-clock", h.Blinds.GetActive)
	mux.HandleFunc("/blinds-clock/start", h.Blinds.Start)
	mux.HandleFunc("/blinds-clock/pause", h.Blinds.Pause)
	mux.HandleFunc("/blinds-clock/resume", h.Blinds.Resume)
	mux.HandleFunc("/blinds-clock/reset", h.Blinds.Reset)
	mux.HandleFunc("/blinds-clock/previous", h.Blinds.PreviousLevel)
	mux.HandleFunc("/blinds-clock/next", h.Blinds.NextLevel)
	mux.HandleFunc("/blinds-clock/levels", h.Blinds.UpdateLevels)

	// ===== PUSH =====
	mux.HandleFunc("/push/config", h.Push.Config)
	mux.HandleFunc("/push/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.Push.Subscribe(w, r)
		case http.MethodDelete:
			h.Push.Unsubscribe(w, r)
		default:
			writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		}
	})

	// ===== PLAYERS =====
	mux.HandleFunc("/players", h.Player.Players)
	mux.HandleFunc("/players/unlinked", h.Account.PublicAvailablePlayers)
	mux.HandleFunc("/players/stats", h.Player.GetPlayerStats)
	mux.HandleFunc("/stats/player", h.Player.GetPlayerStats)

	// ===== STATS =====
	mux.HandleFunc("/stats/sessions", h.Stats.GetStatsSessions)
	mux.HandleFunc("/stats/players", h.Stats.GetStatsPlayers)

	// ===== TEMP DEBUG ADMIN =====
	mux.HandleFunc("/debug/player", h.Debug.DeletePlayer)
	mux.HandleFunc("/debug/player/rename", h.Debug.RenamePlayer)
	mux.HandleFunc("/debug/session", h.Debug.DeleteSession)
	mux.HandleFunc("/debug/session/config", h.Debug.UpdateSessionConfig)
	mux.HandleFunc("/debug/session/finish", h.Debug.DeleteSessionFinish)

	// ===== MIDDLEWARE =====
	var handler http.Handler = mux
	handler = CORSMiddleware(handler)
	handler = RecoveryMiddleware(handler)
	handler = MetricsMiddleware(handler)
	handler = LoggingMiddleware(handler)
	handler = RequestIDMiddleware(handler)

	return handler
}

func isClientRoute(path string) bool {
	return path == "/account" ||
		path == "/blinds" ||
		path == "/blinds/presentation" ||
		len(path) > len("/session/") && path[:len("/session/")] == "/session/" ||
		len(path) > len("/player/") && path[:len("/player/")] == "/player/"
}
