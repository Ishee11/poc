package http

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ishee11/poc/internal/usecase"
)

type TelegramBotSender interface {
	SendLoginConfirmation(ctx context.Context, chatID int64, challenge string, code string) error
	SendNotice(ctx context.Context, chatID int64, text string) error
	AnswerCallback(ctx context.Context, callbackID string, text string) error
	EditDecision(ctx context.Context, chatID int64, messageID int64, code string, approved bool) error
}

type TelegramBotChallengeService interface {
	BotStart(context.Context, string, usecase.TelegramBotUser) (*usecase.TelegramChallengeStatusResult, error)
	BotDecision(context.Context, string, bool, usecase.TelegramBotUser) (*usecase.TelegramChallengeStatusResult, error)
}

type TelegramBotHandler struct {
	service       TelegramBotChallengeService
	sender        TelegramBotSender
	webhookSecret string
}

type telegramUpdate struct {
	Message *struct {
		MessageID int64 `json:"message_id"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From *telegramUser `json:"from"`
		Text string        `json:"text"`
	} `json:"message"`
	CallbackQuery *struct {
		ID      string       `json:"id"`
		From    telegramUser `json:"from"`
		Data    string       `json:"data"`
		Message *struct {
			MessageID int64 `json:"message_id"`
			Chat      struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (u telegramUser) usecaseUser() usecase.TelegramBotUser {
	return usecase.TelegramBotUser{ID: u.ID, Username: u.Username, DisplayName: strings.TrimSpace(u.FirstName + " " + u.LastName)}
}

func (h *TelegramBotHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	provided := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if h == nil || h.service == nil || h.sender == nil || h.webhookSecret == "" ||
		len(provided) != len(h.webhookSecret) || subtle.ConstantTimeCompare([]byte(provided), []byte(h.webhookSecret)) != 1 {
		writeErr(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	defer r.Body.Close()
	var update telegramUpdate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&update); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}
	if update.Message != nil && update.Message.From != nil {
		const start = "/start "
		if strings.HasPrefix(update.Message.Text, start) {
			raw := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, start))
			result, err := h.service.BotStart(r.Context(), raw, update.Message.From.usecaseUser())
			if err == nil {
				err = h.sender.SendLoginConfirmation(r.Context(), update.Message.Chat.ID, raw, result.VerificationCode)
			}
			if err != nil {
				_ = h.sender.SendNotice(r.Context(), update.Message.Chat.ID, "Запрос недействителен или истёк")
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	if update.CallbackQuery != nil {
		parts := strings.SplitN(update.CallbackQuery.Data, ":", 2)
		if len(parts) != 2 || (parts[0] != "approve" && parts[0] != "deny") {
			_ = h.sender.AnswerCallback(r.Context(), update.CallbackQuery.ID, "Запрос недействителен")
			w.WriteHeader(http.StatusOK)
			return
		}
		result, err := h.service.BotDecision(r.Context(), parts[1], parts[0] == "approve", update.CallbackQuery.From.usecaseUser())
		if err != nil {
			_ = h.sender.AnswerCallback(r.Context(), update.CallbackQuery.ID, "Запрос недействителен или уже завершён")
			w.WriteHeader(http.StatusOK)
			return
		}
		approved := parts[0] == "approve"
		_ = h.sender.AnswerCallback(r.Context(), update.CallbackQuery.ID, map[bool]string{true: "Вход подтверждён", false: "Вход отменён"}[approved])
		if update.CallbackQuery.Message != nil {
			_ = h.sender.EditDecision(r.Context(), update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, result.VerificationCode, approved)
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
}
