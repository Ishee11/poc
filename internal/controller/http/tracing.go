package http

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func TracingMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(
		next,
		"http.server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, metricsRouteLabel(r.URL.Path))
		}),
	)
}

func HandlerSpanMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("github.com/ishee11/poc/internal/controller/http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "http.handler")
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
