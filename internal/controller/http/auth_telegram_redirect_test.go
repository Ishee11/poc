package http

import (
	"testing"

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
