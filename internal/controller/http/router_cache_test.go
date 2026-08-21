package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppShellEntryResponsesRequireRevalidation(t *testing.T) {
	router := NewRouter(&Handler{})
	for _, path := range []string{"/", "/profile", "/sw.js", "/manifest.webmanifest"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want 200", path, response.Code)
			}
			if got := response.Header().Get("Cache-Control"); got != "no-cache" {
				t.Fatalf("GET %s Cache-Control = %q, want no-cache", path, got)
			}
		})
	}
}
