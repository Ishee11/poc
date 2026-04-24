package http

import "testing"

func TestMetricsRouteLabel(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "", want: "unknown"},
		{path: "/", want: "/"},
		{path: "/static/css/main.css", want: "/static/*"},
		{path: "/swagger/index.html", want: "/swagger/*"},
		{path: "/session/123", want: "/session/:id"},
		{path: "/player/456", want: "/player/:id"},
		{path: "/stats/sessions", want: "/stats/sessions"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := metricsRouteLabel(tt.path); got != tt.want {
				t.Fatalf("metricsRouteLabel(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
