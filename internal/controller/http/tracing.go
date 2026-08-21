package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

type originalTelegramAuthURLKey struct{}

func TracingMiddleware(next http.Handler) http.Handler {
	restoreURL := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if original, ok := r.Context().Value(originalTelegramAuthURLKey{}).(*url.URL); ok {
			clone := r.Clone(r.Context())
			clone.URL = original
			next.ServeHTTP(w, clone)
			return
		}
		next.ServeHTTP(w, r)
	})
	instrumented := otelhttp.NewHandler(
		restoreURL,
		"http.server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, metricsRouteLabel(r.URL.Path))
		}),
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/auth/telegram/challenge/") || r.URL.Path == "/auth/telegram/callback" {
			original := cloneURL(r.URL)
			ctx := context.WithValue(r.Context(), originalTelegramAuthURLKey{}, original)
			clone := r.Clone(ctx)
			clone.URL = cloneURL(r.URL)
			clone.URL.Path = metricsRouteLabel(r.URL.Path)
			clone.URL.RawPath = ""
			clone.URL.RawQuery = ""
			instrumented.ServeHTTP(w, clone)
			return
		}
		instrumented.ServeHTTP(w, r)
	})
}

func cloneURL(value *url.URL) *url.URL { copy := *value; return &copy }

func HandlerSpanMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("github.com/ishee11/poc/internal/controller/http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "http.handler")
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
