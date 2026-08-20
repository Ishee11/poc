package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthConfigReportsOpenRegistration(t *testing.T) {
	handler := &AuthHandler{cookie: AuthCookieConfig{Enabled: true}}
	request := httptest.NewRequest(http.MethodGet, "/auth/config", nil)
	response := httptest.NewRecorder()

	handler.Config(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body AuthConfigResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Enabled {
		t.Fatal("enabled = false, want true")
	}
	if !body.OpenRegistration {
		t.Fatal("open_registration = false, want true")
	}
}

func TestAuthConfigRejectsUnsafeMethod(t *testing.T) {
	handler := &AuthHandler{cookie: AuthCookieConfig{Enabled: true}}
	request := httptest.NewRequest(http.MethodPost, "/auth/config", nil)
	response := httptest.NewRecorder()

	handler.Config(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
