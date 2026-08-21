package usecase

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type fakeTelegramRepo struct {
	identities map[string]*entity.AuthIdentity
	byUser     map[entity.AuthUserID]*entity.AuthIdentity
	flows      map[string]*entity.AuthOIDCFlow
}

func newFakeTelegramRepo() *fakeTelegramRepo {
	return &fakeTelegramRepo{
		identities: make(map[string]*entity.AuthIdentity),
		byUser:     make(map[entity.AuthUserID]*entity.AuthIdentity),
		flows:      make(map[string]*entity.AuthOIDCFlow),
	}
}

func telegramIdentityKey(provider entity.AuthProvider, subject string) string {
	return string(provider) + ":" + subject
}

func (r *fakeTelegramRepo) SaveIdentity(_ Tx, identity *entity.AuthIdentity) error {
	key := telegramIdentityKey(identity.Provider, identity.Subject)
	if _, ok := r.identities[key]; ok {
		return entity.ErrAuthIdentityLinked
	}
	if _, ok := r.byUser[identity.UserID]; ok {
		return entity.ErrAuthProviderLinked
	}
	copy := *identity
	r.identities[key] = &copy
	r.byUser[identity.UserID] = &copy
	return nil
}

func (r *fakeTelegramRepo) FindIdentity(_ Tx, provider entity.AuthProvider, subject string) (*entity.AuthIdentity, error) {
	identity, ok := r.identities[telegramIdentityKey(provider, subject)]
	if !ok {
		return nil, entity.ErrAuthIdentityNotFound
	}
	copy := *identity
	return &copy, nil
}

func (r *fakeTelegramRepo) ListIdentities(_ Tx, userID entity.AuthUserID) ([]entity.AuthIdentity, error) {
	identity, ok := r.byUser[userID]
	if !ok {
		return []entity.AuthIdentity{}, nil
	}
	return []entity.AuthIdentity{*identity}, nil
}

func (r *fakeTelegramRepo) DeleteIdentity(_ Tx, userID entity.AuthUserID, provider entity.AuthProvider) error {
	identity, ok := r.byUser[userID]
	if !ok || identity.Provider != provider {
		return entity.ErrAuthIdentityNotFound
	}
	delete(r.byUser, userID)
	delete(r.identities, telegramIdentityKey(provider, identity.Subject))
	return nil
}

func (r *fakeTelegramRepo) SaveOIDCFlow(_ Tx, flow *entity.AuthOIDCFlow) error {
	copy := *flow
	r.flows[flow.StateHash] = &copy
	return nil
}

func (r *fakeTelegramRepo) ConsumeOIDCFlow(_ Tx, stateHash string, now time.Time) (*entity.AuthOIDCFlow, error) {
	flow, ok := r.flows[stateHash]
	if !ok || !flow.ExpiresAt.After(now) {
		return nil, entity.ErrOIDCFlowInvalid
	}
	delete(r.flows, stateHash)
	copy := *flow
	return &copy, nil
}

type fakeTelegramOIDCClient struct {
	claims TelegramOIDCClaims
}

func (c fakeTelegramOIDCClient) Exchange(context.Context, string, string, string, string) (TelegramOIDCClaims, error) {
	return c.claims, nil
}

func newTelegramService(repo *fakeTelegramRepo, users *fakeAuthRepo, now time.Time, claims TelegramOIDCClaims) *TelegramAuthService {
	return NewTelegramAuthService(
		repo, repo, users, fakeTxManager{}, fakeAuthUserIDGen{next: "telegram-user"},
		fakeTokenGenerator{token: "flow-token"}, fakeTokenHasher{}, fakeTelegramOIDCClient{claims: claims},
		fakeClock{now: now}, TelegramAuthConfig{
			Enabled: true, ClientID: "123", AuthorizationURL: "https://oauth.telegram.test/auth",
			RedirectURI: "https://poc.test/auth/telegram/callback", FlowTTL: time.Minute,
		},
	)
}

func telegramState(t *testing.T, redirect string) string {
	t.Helper()
	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Query().Get("state")
}

func TestTelegramIdentityLinksToExistingAccountAndLogsIntoSameAccount(t *testing.T) {
	now := time.Date(2026, 8, 21, 3, 0, 0, 0, time.UTC)
	users := newFakeAuthRepo()
	user, err := entity.NewAuthUser("email-user", "user@example.com", "password-hash", entity.AuthRoleUser, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := users.Save(testTx{}, user); err != nil {
		t.Fatal(err)
	}
	repo := newFakeTelegramRepo()
	service := newTelegramService(repo, users, now, TelegramOIDCClaims{Subject: "tg-42", Username: "player42"})

	userID := user.ID
	redirect, err := service.Begin(context.Background(), TelegramBeginCommand{Mode: TelegramOIDCModeLink, UserID: &userID})
	if err != nil {
		t.Fatal(err)
	}
	linked, err := service.Complete(context.Background(), TelegramCompleteCommand{
		State: telegramState(t, redirect), Code: "code", CurrentUserID: &userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if linked.UserID != user.ID || linked.Mode != TelegramOIDCModeLink {
		t.Fatalf("unexpected link result: %+v", linked)
	}

	redirect, err = service.Begin(context.Background(), TelegramBeginCommand{Mode: TelegramOIDCModeLogin})
	if err != nil {
		t.Fatal(err)
	}
	loggedIn, err := service.Complete(context.Background(), TelegramCompleteCommand{State: telegramState(t, redirect), Code: "code"})
	if err != nil {
		t.Fatal(err)
	}
	if loggedIn.UserID != user.ID {
		t.Fatalf("telegram login created or selected another account: got %q want %q", loggedIn.UserID, user.ID)
	}
	if len(users.users) != 1 {
		t.Fatalf("telegram login created duplicate user: %d users", len(users.users))
	}
}

func TestTelegramOnlyAccountCannotRemoveItsLastLoginMethod(t *testing.T) {
	now := time.Date(2026, 8, 21, 3, 0, 0, 0, time.UTC)
	users := newFakeAuthRepo()
	repo := newFakeTelegramRepo()
	service := newTelegramService(repo, users, now, TelegramOIDCClaims{Subject: "tg-new", DisplayName: "New User"})
	redirect, err := service.Begin(context.Background(), TelegramBeginCommand{Mode: TelegramOIDCModeLogin})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Complete(context.Background(), TelegramCompleteCommand{State: telegramState(t, redirect), Code: "code"})
	if err != nil {
		t.Fatal(err)
	}
	if len(users.users) != 1 || result.UserID == "" {
		t.Fatalf("telegram account was not created: %+v", result)
	}
	if err := service.Unlink(context.Background(), result.UserID); !errors.Is(err, entity.ErrLastAuthMethod) {
		t.Fatalf("expected ErrLastAuthMethod, got %v", err)
	}
}

func TestTelegramIdentityCannotBeLinkedToTwoAccounts(t *testing.T) {
	now := time.Date(2026, 8, 21, 3, 0, 0, 0, time.UTC)
	users := newFakeAuthRepo()
	for _, id := range []entity.AuthUserID{"first-user", "second-user"} {
		user, err := entity.NewAuthUser(id, string(id)+"@example.com", "password-hash", entity.AuthRoleUser, now)
		if err != nil {
			t.Fatal(err)
		}
		if err := users.Save(testTx{}, user); err != nil {
			t.Fatal(err)
		}
	}
	repo := newFakeTelegramRepo()
	if err := repo.SaveIdentity(testTx{}, &entity.AuthIdentity{
		Provider: entity.AuthProviderTelegram, Subject: "shared-telegram", UserID: "first-user",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	service := newTelegramService(repo, users, now, TelegramOIDCClaims{Subject: "shared-telegram"})
	second := entity.AuthUserID("second-user")
	redirect, err := service.Begin(context.Background(), TelegramBeginCommand{Mode: TelegramOIDCModeLink, UserID: &second})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Complete(context.Background(), TelegramCompleteCommand{
		State: telegramState(t, redirect), Code: "code", CurrentUserID: &second,
	})
	if !errors.Is(err, entity.ErrAuthIdentityLinked) {
		t.Fatalf("expected ErrAuthIdentityLinked, got %v", err)
	}
}
