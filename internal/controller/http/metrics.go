package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		observed := newObservedResponseWriter(w)

		next.ServeHTTP(observed, r)

		route := metricsRouteLabel(r.URL.Path)
		status := strconv.Itoa(observed.Status())

		labels := []string{r.Method, route, status}
		httpRequestsTotal.WithLabelValues(labels...).Inc()
		httpRequestDuration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
	})
}

func metricsRouteLabel(path string) string {
	switch {
	case path == "":
		return "unknown"
	case path == "/":
		return "/"
	case strings.HasPrefix(path, "/static/"):
		return "/static/*"
	case strings.HasPrefix(path, "/swagger/"):
		return "/swagger/*"
	case strings.HasPrefix(path, "/session/"):
		return "/session/:id"
	case strings.HasPrefix(path, "/player/"):
		return "/player/:id"
	default:
		return path
	}
}
