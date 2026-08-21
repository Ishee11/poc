package http

import (
	"net/http/httptest"
	"testing"
)

func TestTelegramChallengeRequiresSameOrigin(t *testing.T) {
	req := httptest.NewRequest("POST", "https://poc.test/auth/telegram/challenge", nil)
	if sameOriginRequest(req) {
		t.Fatal("missing Origin was accepted")
	}
	req.Header.Set("Origin", "https://evil.test")
	if sameOriginRequest(req) {
		t.Fatal("foreign Origin was accepted")
	}
	req.Header.Set("Origin", "https://poc.test")
	if !sameOriginRequest(req) {
		t.Fatal("same Origin was rejected")
	}
}

func TestTelegramChallengeRouteDoesNotExposeRawTokenToTelemetryLabels(t *testing.T) {
	raw := "/auth/telegram/challenge/raw-secret-must-not-be-logged/status"
	if got := metricsRouteLabel(raw); got != "/auth/telegram/challenge/:token/:action" {
		t.Fatalf("route label=%q", got)
	}
	req := httptest.NewRequest("GET", "https://poc.test"+raw+"?secret=raw", nil)
	if got := safeLoggedQuery(req); got != "" {
		t.Fatalf("logged query=%q", got)
	}
}
