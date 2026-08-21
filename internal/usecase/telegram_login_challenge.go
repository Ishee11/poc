package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

const telegramChallengeTTL = 4 * time.Minute

type TelegramChallengeConfig struct {
	Enabled     bool
	BotUsername string
	TTL         time.Duration
}

type TelegramChallengeCreateResult struct {
	Challenge        string
	BrowserBinding   string
	VerificationCode string
	BotUsername      string
	ExpiresAt        time.Time
}

type TelegramChallengeStatusResult struct {
	Status           entity.TelegramLoginChallengeStatus
	VerificationCode string
	ExpiresAt        time.Time
}

type TelegramBotUser struct {
	ID          int64
	Username    string
	DisplayName string
}

type telegramRateBucket struct {
	window time.Time
	count  int
}
type TelegramChallengeRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]telegramRateBucket
	limit   int
	window  time.Duration
}

func NewTelegramChallengeRateLimiter(limit int, window time.Duration) *TelegramChallengeRateLimiter {
	return &TelegramChallengeRateLimiter{buckets: make(map[string]telegramRateBucket), limit: limit, window: window}
}

func (l *TelegramChallengeRateLimiter) Allow(key string, now time.Time, limit int) bool {
	if l == nil {
		return true
	}
	if limit <= 0 {
		limit = l.limit
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[key]
	if b.window.IsZero() || now.Sub(b.window) >= l.window {
		b = telegramRateBucket{window: now}
	}
	if b.count >= limit {
		return false
	}
	b.count++
	l.buckets[key] = b
	return true
}

type TelegramLoginChallengeService struct {
	repo         TelegramLoginChallengeRepository
	txManager    TxManager
	telegramAuth *TelegramAuthService
	auth         *AuthService
	tokenGen     TokenGenerator
	tokenHash    TokenHasher
	clock        Clock
	config       TelegramChallengeConfig
	limiter      *TelegramChallengeRateLimiter
}

func NewTelegramLoginChallengeService(repo TelegramLoginChallengeRepository, txManager TxManager,
	telegramAuth *TelegramAuthService, auth *AuthService, tokenGen TokenGenerator, tokenHash TokenHasher,
	clock Clock, config TelegramChallengeConfig) *TelegramLoginChallengeService {
	if clock == nil {
		clock = SystemClock{}
	}
	if config.TTL <= 0 {
		config.TTL = telegramChallengeTTL
	}
	return &TelegramLoginChallengeService{repo: repo, txManager: txManager, telegramAuth: telegramAuth,
		auth: auth, tokenGen: tokenGen, tokenHash: tokenHash, clock: clock, config: config,
		limiter: NewTelegramChallengeRateLimiter(30, time.Minute)}
}

func (s *TelegramLoginChallengeService) Enabled() bool {
	return s != nil && s.config.Enabled && s.repo != nil && s.telegramAuth != nil && s.auth != nil && strings.TrimSpace(s.config.BotUsername) != ""
}

func (s *TelegramLoginChallengeService) BotUsername() string {
	if s == nil {
		return ""
	}
	return s.config.BotUsername
}

func (s *TelegramLoginChallengeService) checkRate(kind, key string, limit int) error {
	if key == "" {
		key = "unknown"
	}
	if !s.limiter.Allow(kind+":"+key, s.clock.Now(), limit) {
		return entity.ErrAuthRateLimited
	}
	return nil
}

func (s *TelegramLoginChallengeService) Create(ctx context.Context, rateKey string) (*TelegramChallengeCreateResult, error) {
	if !s.Enabled() {
		return nil, entity.ErrTelegramAuthDisabled
	}
	if err := s.checkRate("create", rateKey, 5); err != nil {
		return nil, err
	}
	challenge, err := s.tokenGen.NewToken()
	if err != nil {
		return nil, err
	}
	binding, err := s.tokenGen.NewToken()
	if err != nil {
		return nil, err
	}
	var digits [2]byte
	if _, err := rand.Read(digits[:]); err != nil {
		return nil, err
	}
	code := fmt.Sprintf("%04d", (int(digits[0])<<8|int(digits[1]))%10000)
	now := s.clock.Now()
	expiresAt := now.Add(s.config.TTL)
	c := &entity.TelegramLoginChallenge{ChallengeHash: s.tokenHash.HashToken(challenge),
		BrowserBindingHash: s.tokenHash.HashToken(binding), VerificationCode: code,
		Status: entity.TelegramLoginChallengePending, CreatedAt: now, ExpiresAt: expiresAt}
	if err := s.txManager.RunInTx(ctx, func(tx Tx) error { return s.repo.SaveTelegramLoginChallenge(tx, c) }); err != nil {
		return nil, err
	}
	return &TelegramChallengeCreateResult{Challenge: challenge, BrowserBinding: binding,
		VerificationCode: code, BotUsername: s.config.BotUsername, ExpiresAt: expiresAt}, nil
}

func (s *TelegramLoginChallengeService) Status(ctx context.Context, raw, binding, rateKey string) (*TelegramChallengeStatusResult, error) {
	if err := s.checkRate("status", rateKey+":"+s.tokenHash.HashToken(raw), 90); err != nil {
		return nil, err
	}
	if raw == "" || binding == "" {
		return nil, entity.ErrTelegramChallengeInvalid
	}
	var c *entity.TelegramLoginChallenge
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		var err error
		c, err = s.repo.FindTelegramLoginChallenge(tx, s.tokenHash.HashToken(raw), s.tokenHash.HashToken(binding), s.clock.Now())
		return err
	})
	if err != nil {
		return nil, err
	}
	return &TelegramChallengeStatusResult{Status: c.Status, VerificationCode: c.VerificationCode, ExpiresAt: c.ExpiresAt}, nil
}

func (s *TelegramLoginChallengeService) BotStart(ctx context.Context, raw string, user TelegramBotUser) (*TelegramChallengeStatusResult, error) {
	if err := s.checkRate("bot", fmt.Sprint(user.ID), 20); err != nil {
		return nil, err
	}
	if raw == "" || user.ID == 0 {
		return nil, entity.ErrTelegramChallengeInvalid
	}
	var c *entity.TelegramLoginChallenge
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		var err error
		c, err = s.repo.ClaimTelegramLoginChallengeActor(tx, s.tokenHash.HashToken(raw), fmt.Sprint(user.ID), s.clock.Now())
		return err
	})
	if err != nil || c.Status != entity.TelegramLoginChallengePending {
		if err != nil {
			return nil, err
		}
		return nil, entity.ErrTelegramChallengeState
	}
	return &TelegramChallengeStatusResult{Status: c.Status, VerificationCode: c.VerificationCode, ExpiresAt: c.ExpiresAt}, nil
}

func (s *TelegramLoginChallengeService) BotDecision(ctx context.Context, raw string, approve bool, user TelegramBotUser) (*TelegramChallengeStatusResult, error) {
	if err := s.checkRate("bot", fmt.Sprint(user.ID), 20); err != nil {
		return nil, err
	}
	if raw == "" || user.ID == 0 {
		return nil, entity.ErrTelegramChallengeInvalid
	}
	subject := fmt.Sprint(user.ID)
	var c *entity.TelegramLoginChallenge
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		current, err := s.repo.FindTelegramLoginChallengeForBot(tx, s.tokenHash.HashToken(raw), s.clock.Now())
		if err != nil {
			return err
		}
		if current.TelegramSubject != "" && current.TelegramSubject != subject {
			return entity.ErrTelegramChallengeActor
		}
		if approve {
			c, err = s.repo.ApproveTelegramLoginChallenge(tx, s.tokenHash.HashToken(raw), subject, user.Username, user.DisplayName, s.clock.Now())
		} else {
			c, err = s.repo.DenyTelegramLoginChallenge(tx, s.tokenHash.HashToken(raw), subject, s.clock.Now())
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &TelegramChallengeStatusResult{Status: c.Status, VerificationCode: c.VerificationCode, ExpiresAt: c.ExpiresAt}, nil
}

func (s *TelegramLoginChallengeService) Cancel(ctx context.Context, raw, binding, rateKey string) error {
	if err := s.checkRate("cancel", rateKey+":"+s.tokenHash.HashToken(raw), 10); err != nil {
		return err
	}
	if raw == "" || binding == "" {
		return entity.ErrTelegramChallengeInvalid
	}
	return s.txManager.RunInTx(ctx, func(tx Tx) error {
		c, err := s.repo.LockTelegramLoginChallenge(tx, s.tokenHash.HashToken(raw), s.tokenHash.HashToken(binding), s.clock.Now())
		if err != nil {
			return err
		}
		if c.Status == entity.TelegramLoginChallengeDenied {
			return nil
		}
		if c.Status != entity.TelegramLoginChallengePending {
			return entity.ErrTelegramChallengeState
		}
		_, err = s.repo.DenyTelegramLoginChallenge(tx, s.tokenHash.HashToken(raw), "browser", s.clock.Now())
		return err
	})
}

func (s *TelegramLoginChallengeService) Complete(ctx context.Context, raw, binding, userAgent, ip string) (*LoginResult, error) {
	if err := s.checkRate("complete", ip+":"+s.tokenHash.HashToken(raw), 10); err != nil {
		return nil, err
	}
	if raw == "" || binding == "" {
		return nil, entity.ErrTelegramChallengeInvalid
	}
	var result *LoginResult
	err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		c, err := s.repo.LockTelegramLoginChallenge(tx, s.tokenHash.HashToken(raw), s.tokenHash.HashToken(binding), s.clock.Now())
		if err != nil {
			return err
		}
		if c.Status != entity.TelegramLoginChallengeApproved || c.TelegramSubject == "" {
			return entity.ErrTelegramChallengeState
		}
		userID, err := s.telegramAuth.resolveOrCreateTelegramIdentity(tx, TelegramOIDCClaims{Subject: c.TelegramSubject, Username: c.TelegramUsername, DisplayName: c.TelegramName})
		if err != nil {
			return err
		}
		result, err = s.auth.loginUserInTx(tx, userID, userAgent, ip)
		if err != nil {
			return err
		}
		return s.repo.ConsumeTelegramLoginChallenge(tx, c.ChallengeHash, s.clock.Now())
	})
	return result, err
}
