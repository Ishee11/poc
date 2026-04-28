package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
)

type BlindClockPushConfig struct {
	Enabled   bool
	PublicKey string
}

type BlindClockPushClientConfig struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key,omitempty"`
}

type BlindClockPushSubscriptionStatus struct {
	Subscribed      bool `json:"subscribed"`
	NotifyWarning60 bool `json:"notify_warning_60"`
	NotifyWarning10 bool `json:"notify_warning_10"`
}

type BlindClockPushSubscriptionInput struct {
	Endpoint        string `json:"endpoint"`
	KeyAuth         string `json:"key_auth"`
	KeyP256DH       string `json:"key_p256dh"`
	UserAgent       string `json:"user_agent"`
	NotifyWarning60 bool   `json:"notify_warning_60"`
	NotifyWarning10 bool   `json:"notify_warning_10"`
}

type BlindClockPushService struct {
	repo   BlindClockPushRepository
	sender BlindClockPushSender
	cfg    BlindClockPushConfig
}

type BlindClockPushTestResult struct {
	Subscriptions int      `json:"subscriptions"`
	Delivered     int      `json:"delivered"`
	Failed        int      `json:"failed"`
	Errors        []string `json:"errors,omitempty"`
}

type BlindClockPushSender interface {
	SendTest(subscription entity.BlindClockPushSubscription) error
}

func NewBlindClockPushService(
	repo BlindClockPushRepository,
	sender BlindClockPushSender,
	cfg BlindClockPushConfig,
) *BlindClockPushService {
	return &BlindClockPushService{
		repo:   repo,
		sender: sender,
		cfg:    cfg,
	}
}

func (s *BlindClockPushService) GetClientConfig() BlindClockPushClientConfig {
	if !s.cfg.Enabled {
		return BlindClockPushClientConfig{Enabled: false}
	}

	return BlindClockPushClientConfig{
		Enabled:   true,
		PublicKey: s.cfg.PublicKey,
	}
}

func (s *BlindClockPushService) Subscribe(input BlindClockPushSubscriptionInput) error {
	if !s.cfg.Enabled {
		return entity.ErrPushDisabled
	}
	if strings.TrimSpace(input.Endpoint) == "" ||
		strings.TrimSpace(input.KeyAuth) == "" ||
		strings.TrimSpace(input.KeyP256DH) == "" {
		return entity.ErrInvalidPushSubscription
	}

	now := time.Now()
	subscription := entity.BlindClockPushSubscription{
		Endpoint:        strings.TrimSpace(input.Endpoint),
		KeyAuth:         strings.TrimSpace(input.KeyAuth),
		KeyP256DH:       strings.TrimSpace(input.KeyP256DH),
		UserAgent:       strings.TrimSpace(input.UserAgent),
		NotifyWarning60: input.NotifyWarning60,
		NotifyWarning10: input.NotifyWarning10,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.UpsertSubscription(subscription); err != nil {
		return err
	}
	if s.sender != nil {
		_ = s.sender.SendTest(subscription)
	}

	return nil
}

func (s *BlindClockPushService) Unsubscribe(endpoint string) error {
	if !s.cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(endpoint) == "" {
		return entity.ErrInvalidPushSubscription
	}

	return s.repo.DeleteSubscription(strings.TrimSpace(endpoint))
}

func (s *BlindClockPushService) GetSubscriptionStatus(endpoint string) (BlindClockPushSubscriptionStatus, error) {
	status := BlindClockPushSubscriptionStatus{
		Subscribed:      false,
		NotifyWarning60: true,
		NotifyWarning10: true,
	}
	if !s.cfg.Enabled {
		return status, nil
	}
	if strings.TrimSpace(endpoint) == "" {
		return status, entity.ErrInvalidPushSubscription
	}

	subscription, err := s.repo.GetSubscription(strings.TrimSpace(endpoint))
	if err != nil {
		if err == pgx.ErrNoRows {
			return status, nil
		}
		return status, err
	}

	return BlindClockPushSubscriptionStatus{
		Subscribed:      true,
		NotifyWarning60: subscription.NotifyWarning60,
		NotifyWarning10: subscription.NotifyWarning10,
	}, nil
}

func (s *BlindClockPushService) SendTestToAll() (BlindClockPushTestResult, error) {
	result := BlindClockPushTestResult{}
	if !s.cfg.Enabled {
		return result, entity.ErrPushDisabled
	}
	if s.sender == nil {
		return result, entity.ErrPushDisabled
	}

	subscriptions, err := s.repo.ListSubscriptions()
	if err != nil {
		return result, err
	}

	result.Subscriptions = len(subscriptions)
	for _, subscription := range subscriptions {
		if err := s.sender.SendTest(subscription); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", subscription.Endpoint, err))
			continue
		}

		result.Delivered++
	}

	return result, nil
}
