package usecase

import (
	"strings"
	"time"

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

type BlindClockPushSubscriptionInput struct {
	Endpoint  string `json:"endpoint"`
	KeyAuth   string `json:"key_auth"`
	KeyP256DH string `json:"key_p256dh"`
	UserAgent string `json:"user_agent"`
}

type BlindClockPushService struct {
	repo BlindClockPushRepository
	cfg  BlindClockPushConfig
}

func NewBlindClockPushService(repo BlindClockPushRepository, cfg BlindClockPushConfig) *BlindClockPushService {
	return &BlindClockPushService{
		repo: repo,
		cfg:  cfg,
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
	return s.repo.UpsertSubscription(entity.BlindClockPushSubscription{
		Endpoint:  strings.TrimSpace(input.Endpoint),
		KeyAuth:   strings.TrimSpace(input.KeyAuth),
		KeyP256DH: strings.TrimSpace(input.KeyP256DH),
		UserAgent: strings.TrimSpace(input.UserAgent),
		CreatedAt: now,
		UpdatedAt: now,
	})
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
