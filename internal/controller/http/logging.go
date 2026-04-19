package http

import (
	"log/slog"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		observed := newObservedResponseWriter(w)

		next.ServeHTTP(observed, r)

		status := observed.Status()
		level := slog.LevelInfo
		if status >= http.StatusInternalServerError {
			level = slog.LevelError
		} else if status >= http.StatusBadRequest {
			level = slog.LevelWarn
		}

		attrs := []any{
			"request_id", GetRequestID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", status,
			"duration_ms", float64(time.Since(start).Microseconds()) / 1000,
			"bytes", observed.Bytes(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		}
		if observed.ErrorCode() != "" {
			attrs = append(attrs, "error_code", observed.ErrorCode())
		}

		slog.Log(
			r.Context(),
			level,
			"http_request",
			attrs...,
		)
	})
}
