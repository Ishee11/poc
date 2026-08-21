package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type fakeBotChallengeService struct {
	startErr    error
	decisionErr error
	raw         string
	approved    bool
	user        usecase.TelegramBotUser
}

func (s *fakeBotChallengeService) BotStart(_ context.Context, raw string, user usecase.TelegramBotUser) (*usecase.TelegramChallengeStatusResult, error) {
	s.raw, s.user = raw, user
	if s.startErr != nil {
		return nil, s.startErr
	}
	return &usecase.TelegramChallengeStatusResult{Status: entity.TelegramLoginChallengePending, VerificationCode: "4831", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (s *fakeBotChallengeService) BotDecision(_ context.Context, raw string, approved bool, user usecase.TelegramBotUser) (*usecase.TelegramChallengeStatusResult, error) {
	s.raw, s.approved, s.user = raw, approved, user
	if s.decisionErr != nil {
		return nil, s.decisionErr
	}
	return &usecase.TelegramChallengeStatusResult{Status: entity.TelegramLoginChallengeApproved, VerificationCode: "4831"}, nil
}

type fakeBotSender struct {
	confirmation bool
	notice       bool
	answered     bool
	edited       bool
}

func (s *fakeBotSender) SendLoginConfirmation(context.Context, int64, string, string) error {
	s.confirmation = true
	return nil
}
func (s *fakeBotSender) SendNotice(context.Context, int64, string) error { s.notice = true; return nil }
func (s *fakeBotSender) AnswerCallback(context.Context, string, string) error {
	s.answered = true
	return nil
}
func (s *fakeBotSender) EditDecision(context.Context, int64, int64, string, bool) error {
	s.edited = true
	return nil
}

func botWebhookRequest(body, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/telegram/login-bot/webhook", strings.NewReader(body))
	if secret != "" {
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	}
	return req
}

func TestTelegramBotWebhookRequiresSecret(t *testing.T) {
	h := &TelegramBotHandler{service: &fakeBotChallengeService{}, sender: &fakeBotSender{}, webhookSecret: "expected"}
	w := httptest.NewRecorder()
	h.Webhook(w, botWebhookRequest(`{}`, "wrong"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestTelegramBotWebhookStartAndApprove(t *testing.T) {
	service := &fakeBotChallengeService{}
	sender := &fakeBotSender{}
	h := &TelegramBotHandler{service: service, sender: sender, webhookSecret: "secret"}
	start := `{"message":{"message_id":1,"chat":{"id":10},"from":{"id":42,"username":"poker"},"text":"/start high-entropy"}}`
	w := httptest.NewRecorder()
	h.Webhook(w, botWebhookRequest(start, "secret"))
	if w.Code != http.StatusOK || service.raw != "high-entropy" || service.user.ID != 42 || !sender.confirmation {
		t.Fatalf("start status=%d service=%+v sender=%+v", w.Code, service, sender)
	}

	callback := `{"callback_query":{"id":"cb","from":{"id":42},"data":"approve:high-entropy","message":{"message_id":2,"chat":{"id":10}}}}`
	w = httptest.NewRecorder()
	h.Webhook(w, botWebhookRequest(callback, "secret"))
	if w.Code != http.StatusOK || !service.approved || !sender.answered || !sender.edited {
		t.Fatalf("callback service=%+v sender=%+v", service, sender)
	}
}

func TestTelegramBotWebhookInvalidStartAndCancel(t *testing.T) {
	service := &fakeBotChallengeService{startErr: errors.New("expired")}
	sender := &fakeBotSender{}
	h := &TelegramBotHandler{service: service, sender: sender, webhookSecret: "secret"}
	start := `{"message":{"chat":{"id":10},"from":{"id":42},"text":"/start expired"}}`
	w := httptest.NewRecorder()
	h.Webhook(w, botWebhookRequest(start, "secret"))
	if !sender.notice {
		t.Fatal("invalid start did not produce notice")
	}
	service.startErr = nil
	callback := `{"callback_query":{"id":"cb","from":{"id":42},"data":"deny:high-entropy"}}`
	w = httptest.NewRecorder()
	h.Webhook(w, botWebhookRequest(callback, "secret"))
	if service.approved {
		t.Fatal("deny callback was treated as approve")
	}
}
