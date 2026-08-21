package infra

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type telegramBotRoundTripper func(*http.Request) (*http.Response, error)

func (f telegramBotRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTelegramBotConfirmationUsesInlineChallengeActions(t *testing.T) {
	client := NewTelegramBotClient("backend-only-token")
	var body string
	client.httpClient.Transport = telegramBotRoundTripper(func(r *http.Request) (*http.Response, error) {
		payload, _ := io.ReadAll(r.Body)
		body = string(payload)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})
	if err := client.SendLoginConfirmation(context.Background(), 42, "high-entropy", "4831"); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`Код: 4831`, `approve:high-entropy`, `deny:high-entropy`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("payload missing %q: %s", expected, body)
		}
	}
}
