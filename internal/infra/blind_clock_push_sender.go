package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/ishee11/poc/internal/entity"
)

type BlindClockPushSender struct {
	subject    string
	publicKey  string
	privateKey string
}

func NewBlindClockPushSender(subject, publicKey, privateKey string) *BlindClockPushSender {
	if subject == "" || publicKey == "" || privateKey == "" {
		return nil
	}

	return &BlindClockPushSender{
		subject:    subject,
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func (s *BlindClockPushSender) SendTest(subscription entity.BlindClockPushSubscription) error {
	payload, err := json.Marshal(map[string]any{
		"kind":  "test",
		"title": "Blind timer alerts enabled",
		"body":  "This is a test notification from semenovv.space.",
		"tag":   "blind-clock-test",
		"url":   "/blinds/presentation",
	})
	if err != nil {
		return err
	}

	resp, err := webpush.SendNotificationWithContext(context.Background(), payload, &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			Auth:   subscription.KeyAuth,
			P256dh: subscription.KeyP256DH,
		},
	}, &webpush.Options{
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
		TTL:             120,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if len(body) > 0 {
			return fmt.Errorf("push test failed with status %d: %s", resp.StatusCode, string(body))
		}
		return fmt.Errorf("push test failed with status %d", resp.StatusCode)
	}

	return nil
}
