package http

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(
					r.Context(),
					"panic_recovered",
					"request_id", GetRequestID(r.Context()),
					"panic", recovered,
					"stack", string(debug.Stack()),
				)

				if observed, ok := w.(*observedResponseWriter); ok && observed.status != 0 {
					return
				}

				writeErr(w, r, http.StatusInternalServerError, "internal_error", nil)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
