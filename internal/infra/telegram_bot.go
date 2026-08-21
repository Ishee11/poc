package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TelegramBotClient struct {
	token      string
	httpClient *http.Client
}

func NewTelegramBotClient(token string) *TelegramBotClient {
	return &TelegramBotClient{token: token, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *TelegramBotClient) call(ctx context.Context, method string, payload any) error {
	if c == nil || c.token == "" {
		return fmt.Errorf("telegram bot disabled")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+c.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram bot api status %d", resp.StatusCode)
	}
	return nil
}

func (c *TelegramBotClient) SendLoginConfirmation(ctx context.Context, chatID int64, challenge, code string) error {
	return c.call(ctx, "sendMessage", map[string]any{
		"chat_id": chatID,
		"text":    "Подтвердить вход в Poker?\n\nКод: " + code,
		"reply_markup": map[string]any{"inline_keyboard": [][]map[string]string{{
			{"text": "Подтвердить", "callback_data": "approve:" + challenge},
			{"text": "Отмена", "callback_data": "deny:" + challenge},
		}}},
	})
}

func (c *TelegramBotClient) SendNotice(ctx context.Context, chatID int64, text string) error {
	return c.call(ctx, "sendMessage", map[string]any{"chat_id": chatID, "text": text})
}

func (c *TelegramBotClient) AnswerCallback(ctx context.Context, callbackID, text string) error {
	if callbackID == "" {
		return nil
	}
	return c.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": callbackID, "text": text})
}

func (c *TelegramBotClient) EditDecision(ctx context.Context, chatID, messageID int64, code string, approved bool) error {
	decision := "Вход отменён"
	if approved {
		decision = "Вход подтверждён"
	}
	return c.call(ctx, "editMessageText", map[string]any{"chat_id": chatID, "message_id": messageID,
		"text": decision + "\n\nКод: " + code})
}
