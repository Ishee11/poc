package http

import (
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf(
			"%s %s duration=%s request_id=%s",
			r.Method,
			r.URL.Path,
			time.Since(start),
			GetRequestID(r.Context()),
		)
	})
}
