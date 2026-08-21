package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func TestTelegramCallbackSuccessRedirectUsesProfileScreen(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{name: "login", mode: usecase.TelegramOIDCModeLogin, want: "/profile?telegram=logged_in"},
		{name: "link", mode: usecase.TelegramOIDCModeLink, want: "/profile?telegram=linked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := telegramCallbackSuccessRedirect(tt.mode); got != tt.want {
				t.Fatalf("telegram redirect = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTelegramAuthFailureRedirectUsesProfileAndControlledCategory(t *testing.T) {
	tests := []struct {
		category string
		want     string
	}{
		{category: "cancelled", want: "/profile?telegram_error=cancelled"},
		{category: "provider_unavailable", want: "/profile?telegram_error=provider_unavailable"},
		{category: "disabled", want: "/profile?telegram_error=disabled"},
		{category: "unexpected", want: "/profile?telegram_error=failed"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			if got := telegramAuthFailureRedirect(tt.category); got != tt.want {
				t.Fatalf("telegram failure redirect = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTelegramAuthErrorCategoryDoesNotExposeRawErrors(t *testing.T) {
	if got := telegramAuthErrorCategory(entity.ErrTelegramAuthDisabled); got != "disabled" {
		t.Fatalf("disabled category = %q", got)
	}
	if got := telegramAuthErrorCategory(entity.ErrTelegramProviderUnavailable); got != "provider_unavailable" {
		t.Fatalf("provider category = %q", got)
	}
	if got := telegramAuthErrorCategory(errors.New("secret provider response")); got != "failed" {
		t.Fatalf("fallback category = %q", got)
	}
}

func TestTelegramStartDisabledRedirectsWithoutPanic(t *testing.T) {
	handler := &AuthHandler{}
	request := httptest.NewRequest(http.MethodGet, "/auth/telegram/start?mode=login", nil)
	response := httptest.NewRecorder()

	handler.TelegramStart(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}
	if got := response.Header().Get("Location"); got != "/profile?telegram_error=disabled" {
		t.Fatalf("location = %q", got)
	}
}
