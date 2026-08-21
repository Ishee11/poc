package usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type challengeTokenGen struct{ n int }

func (g *challengeTokenGen) NewToken() (string, error) {
	g.n++
	return fmt.Sprintf("random-token-%d-with-at-least-128-bits", g.n), nil
}

type mutableChallengeClock struct{ now time.Time }

func (c *mutableChallengeClock) Now() time.Time { return c.now }

type serialChallengeTxManager struct{ mu sync.Mutex }

func (m *serialChallengeTxManager) RunInTx(_ context.Context, fn func(Tx) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(testTx{})
}

type fakeChallengeRepo struct {
	rows map[string]*entity.TelegramLoginChallenge
}

func newFakeChallengeRepo() *fakeChallengeRepo {
	return &fakeChallengeRepo{rows: map[string]*entity.TelegramLoginChallenge{}}
}
func cloneChallenge(c *entity.TelegramLoginChallenge) *entity.TelegramLoginChallenge {
	copy := *c
	return &copy
}

func (r *fakeChallengeRepo) SaveTelegramLoginChallenge(_ Tx, c *entity.TelegramLoginChallenge) error {
	r.rows[c.ChallengeHash] = cloneChallenge(c)
	return nil
}
func (r *fakeChallengeRepo) find(hash, binding string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	c, ok := r.rows[hash]
	if !ok || (binding != "" && c.BrowserBindingHash != binding) {
		return nil, entity.ErrTelegramChallengeInvalid
	}
	if !c.ExpiresAt.After(now) && c.Status != entity.TelegramLoginChallengeConsumed {
		c.Status = entity.TelegramLoginChallengeExpired
	}
	return cloneChallenge(c), nil
}
func (r *fakeChallengeRepo) FindTelegramLoginChallenge(_ Tx, h, b string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	return r.find(h, b, now)
}
func (r *fakeChallengeRepo) FindTelegramLoginChallengeForBot(_ Tx, h string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	return r.find(h, "", now)
}
func (r *fakeChallengeRepo) ClaimTelegramLoginChallengeActor(_ Tx, h, subject string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	c, err := r.find(h, "", now)
	if err != nil {
		return nil, err
	}
	if c.Status != entity.TelegramLoginChallengePending || (c.TelegramSubject != "" && c.TelegramSubject != subject) {
		return nil, entity.ErrTelegramChallengeActor
	}
	r.rows[h].TelegramSubject = subject
	return cloneChallenge(r.rows[h]), nil
}
func (r *fakeChallengeRepo) ApproveTelegramLoginChallenge(_ Tx, h, subject, username, name string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	c, err := r.find(h, "", now)
	if err != nil {
		return nil, err
	}
	if c.Status == entity.TelegramLoginChallengeApproved && c.TelegramSubject == subject {
		return c, nil
	}
	if c.Status != entity.TelegramLoginChallengePending || (c.TelegramSubject != "" && c.TelegramSubject != subject) {
		return nil, entity.ErrTelegramChallengeState
	}
	row := r.rows[h]
	row.Status = entity.TelegramLoginChallengeApproved
	row.TelegramSubject = subject
	row.TelegramUsername = username
	row.TelegramName = name
	row.ApprovedAt = &now
	return cloneChallenge(row), nil
}
func (r *fakeChallengeRepo) DenyTelegramLoginChallenge(_ Tx, h, subject string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	c, err := r.find(h, "", now)
	if err != nil {
		return nil, err
	}
	if c.Status == entity.TelegramLoginChallengeDenied && c.TelegramSubject == subject {
		return c, nil
	}
	if c.Status != entity.TelegramLoginChallengePending {
		return nil, entity.ErrTelegramChallengeState
	}
	row := r.rows[h]
	row.Status = entity.TelegramLoginChallengeDenied
	row.TelegramSubject = subject
	return cloneChallenge(row), nil
}
func (r *fakeChallengeRepo) LockTelegramLoginChallenge(_ Tx, h, b string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	return r.find(h, b, now)
}
func (r *fakeChallengeRepo) ConsumeTelegramLoginChallenge(_ Tx, h string, now time.Time) error {
	c := r.rows[h]
	if c == nil || c.Status != entity.TelegramLoginChallengeApproved || !c.ExpiresAt.After(now) {
		return entity.ErrTelegramChallengeState
	}
	c.Status = entity.TelegramLoginChallengeConsumed
	c.ConsumedAt = &now
	return nil
}

func newChallengeService(t *testing.T, now time.Time) (*TelegramLoginChallengeService, *fakeChallengeRepo, *fakeAuthRepo, *mutableChallengeClock) {
	t.Helper()
	challengeRepo := newFakeChallengeRepo()
	users := newFakeAuthRepo()
	identities := newFakeTelegramRepo()
	clock := &mutableChallengeClock{now: now}
	txManager := &serialChallengeTxManager{}
	telegram := NewTelegramAuthService(identities, identities, users, txManager, fakeAuthUserIDGen{next: "telegram-user"},
		&challengeTokenGen{}, fakeTokenHasher{}, fakeTelegramOIDCClient{}, clock,
		TelegramAuthConfig{Enabled: true, ClientID: "oidc", RedirectURI: "https://poc.test/auth/telegram/callback"})
	auth := NewAuthService(users, users, users, txManager, fakeAuthSessionIDGen{next: "challenge-session"},
		&fakeLoginAttemptIDGen{}, fakeTokenGenerator{token: "session-token"}, fakeTokenHasher{}, fakePasswordVerifier{}, clock,
		AuthPolicy{SessionTTL: time.Hour, IdleTTL: time.Hour, RateLimitWindow: time.Minute, MaxFailedAttempts: 5})
	service := NewTelegramLoginChallengeService(challengeRepo, txManager, telegram, auth,
		&challengeTokenGen{}, fakeTokenHasher{}, clock, TelegramChallengeConfig{Enabled: true, BotUsername: "PokerLoginBot", TTL: 4 * time.Minute})
	return service, challengeRepo, users, clock
}

func TestTelegramChallengeCreationTTLAndBrowserBinding(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	service, _, _, _ := newChallengeService(t, now)
	created, err := service.Create(context.Background(), "ip-1")
	if err != nil {
		t.Fatal(err)
	}
	if created.Challenge == created.BrowserBinding || created.ExpiresAt.Sub(now) != 4*time.Minute || len(created.VerificationCode) != 4 {
		t.Fatalf("unexpected challenge: %+v", created)
	}
	if _, err := service.Status(context.Background(), created.Challenge, "other-browser", "ip-2"); !errors.Is(err, entity.ErrTelegramChallengeInvalid) {
		t.Fatalf("other browser status: %v", err)
	}
	status, err := service.Status(context.Background(), created.Challenge, created.BrowserBinding, "ip-1")
	if err != nil || status.Status != entity.TelegramLoginChallengePending {
		t.Fatalf("status=%+v err=%v", status, err)
	}
}

func TestTelegramChallengeApproveCompletesOnceAndReusesIdentity(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	service, _, users, _ := newChallengeService(t, now)
	existing, _ := entity.NewAuthUser("existing-user", "existing@example.test", "hash", entity.AuthRoleUser, now)
	_ = users.Save(testTx{}, existing)
	identities := service.telegramAuth.identities.(*fakeTelegramRepo)
	_ = identities.SaveIdentity(testTx{}, &entity.AuthIdentity{Provider: entity.AuthProviderTelegram, Subject: "42", UserID: existing.ID, CreatedAt: now, UpdatedAt: now})
	created, _ := service.Create(context.Background(), "ip")
	bot := TelegramBotUser{ID: 42, Username: "existing", DisplayName: "Existing"}
	if _, err := service.BotStart(context.Background(), created.Challenge, bot); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BotDecision(context.Background(), created.Challenge, true, bot); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BotDecision(context.Background(), created.Challenge, true, bot); err != nil {
		t.Fatalf("repeat approve not idempotent: %v", err)
	}
	result, err := service.Complete(context.Background(), created.Challenge, created.BrowserBinding, "ua", "ip")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token != "session-token" || result.User.UserID != existing.ID || len(users.users) != 1 || len(users.sessions) != 1 {
		t.Fatalf("result=%+v users=%d sessions=%d", result, len(users.users), len(users.sessions))
	}
	if _, err := service.Complete(context.Background(), created.Challenge, created.BrowserBinding, "ua", "ip"); !errors.Is(err, entity.ErrTelegramChallengeState) {
		t.Fatalf("replay complete: %v", err)
	}
}

func TestTelegramChallengeConcurrentCompleteCreatesOneResult(t *testing.T) {
	service, _, _, _ := newChallengeService(t, time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC))
	created, _ := service.Create(context.Background(), "ip")
	bot := TelegramBotUser{ID: 77}
	_, _ = service.BotStart(context.Background(), created.Challenge, bot)
	_, _ = service.BotDecision(context.Background(), created.Challenge, true, bot)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := service.Complete(context.Background(), created.Challenge, created.BrowserBinding, "ua", "ip")
			errs <- err
		}()
	}
	success := 0
	for i := 0; i < 2; i++ {
		if err := <-errs; err == nil {
			success++
		} else if !errors.Is(err, entity.ErrTelegramChallengeState) {
			t.Fatalf("unexpected race error: %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("successful completions=%d", success)
	}
}

func TestTelegramChallengeFirstLoginCreatesEquivalentTelegramAccount(t *testing.T) {
	service, _, users, _ := newChallengeService(t, time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC))
	created, _ := service.Create(context.Background(), "ip")
	bot := TelegramBotUser{ID: 99, Username: "first_login"}
	_, _ = service.BotStart(context.Background(), created.Challenge, bot)
	_, _ = service.BotDecision(context.Background(), created.Challenge, true, bot)
	result, err := service.Complete(context.Background(), created.Challenge, created.BrowserBinding, "ua", "ip")
	if err != nil {
		t.Fatal(err)
	}
	user := users.users[result.User.UserID]
	if user == nil || user.PasswordHash != telegramPasswordHash || len(users.users) != 1 {
		t.Fatalf("first login user=%+v", user)
	}
}

func TestTelegramChallengeDenyExpiryActorAndRateLimit(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	service, _, _, clock := newChallengeService(t, now)
	created, _ := service.Create(context.Background(), "deny-ip")
	owner := TelegramBotUser{ID: 1}
	other := TelegramBotUser{ID: 2}
	if _, err := service.BotStart(context.Background(), created.Challenge, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BotDecision(context.Background(), created.Challenge, true, other); !errors.Is(err, entity.ErrTelegramChallengeActor) {
		t.Fatalf("actor mismatch: %v", err)
	}
	if _, err := service.BotDecision(context.Background(), created.Challenge, false, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BotDecision(context.Background(), created.Challenge, false, owner); err != nil {
		t.Fatalf("repeat deny not idempotent: %v", err)
	}
	if _, err := service.Complete(context.Background(), created.Challenge, created.BrowserBinding, "ua", "ip"); !errors.Is(err, entity.ErrTelegramChallengeState) {
		t.Fatalf("denied complete: %v", err)
	}

	expiring, _ := service.Create(context.Background(), "expire-ip")
	clock.now = now.Add(5 * time.Minute)
	if _, err := service.BotStart(context.Background(), expiring.Challenge, TelegramBotUser{ID: 3}); err == nil {
		t.Fatal("expired challenge accepted")
	}
	if _, err := service.BotStart(context.Background(), "unknown", TelegramBotUser{ID: 4}); !errors.Is(err, entity.ErrTelegramChallengeInvalid) {
		t.Fatalf("unknown challenge: %v", err)
	}

	service.limiter = NewTelegramChallengeRateLimiter(30, time.Minute)
	clock.now = now
	for i := 0; i < 5; i++ {
		if _, err := service.Create(context.Background(), "limited-ip"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.Create(context.Background(), "limited-ip"); !errors.Is(err, entity.ErrAuthRateLimited) {
		t.Fatalf("rate limit: %v", err)
	}
}
