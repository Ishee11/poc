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

func (r *fakeTelegramRepo) ReplaceIdentitySubject(_ Tx, provider entity.AuthProvider, oldSubject string, identity *entity.AuthIdentity) error {
	oldKey := telegramIdentityKey(provider, oldSubject)
	old, ok := r.identities[oldKey]
	if !ok || old.UserID != identity.UserID {
		return entity.ErrAuthIdentityNotFound
	}
	newKey := telegramIdentityKey(provider, identity.Subject)
	if current, exists := r.identities[newKey]; exists && current.UserID != identity.UserID {
		return entity.ErrAuthIdentityLinked
	}
	copy := *identity
	delete(r.identities, oldKey)
	r.identities[newKey] = &copy
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
	err    error
}

func (c fakeTelegramOIDCClient) Exchange(context.Context, string, string, string, string) (TelegramOIDCClaims, error) {
	return c.claims, c.err
}

func TestTelegramProviderUnavailableRemainsRecoverable(t *testing.T) {
	now := time.Date(2026, 8, 21, 3, 0, 0, 0, time.UTC)
	users := newFakeAuthRepo()
	repo := newFakeTelegramRepo()
	service := newTelegramService(repo, users, now, TelegramOIDCClaims{Subject: "unused"})
	redirect, err := service.Begin(context.Background(), TelegramBeginCommand{Mode: TelegramOIDCModeLogin})
	if err != nil {
		t.Fatal(err)
	}
	service.client = fakeTelegramOIDCClient{err: entity.ErrTelegramProviderUnavailable}

	_, err = service.Complete(context.Background(), TelegramCompleteCommand{
		State: telegramState(t, redirect), Code: "code",
	})
	if !errors.Is(err, entity.ErrTelegramProviderUnavailable) {
		t.Fatalf("expected provider unavailable, got %v", err)
	}
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

func TestTelegramOIDCLegacySubjectMigratesToCanonicalUserID(t *testing.T) {
	now := time.Date(2026, 8, 21, 3, 0, 0, 0, time.UTC)
	users := newFakeAuthRepo()
	user, err := entity.NewAuthUser("email-user", "ishee@yandex.ru", "password-hash", entity.AuthRoleUser, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := users.Save(testTx{}, user); err != nil {
		t.Fatal(err)
	}
	repo := newFakeTelegramRepo()
	if err := repo.SaveIdentity(testTx{}, &entity.AuthIdentity{
		Provider: entity.AuthProviderTelegram, Subject: "oidc-subject", UserID: user.ID,
		Username: "semenovv", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	service := newTelegramService(repo, users, now, TelegramOIDCClaims{
		Subject: "42", LegacySubject: "oidc-subject", Username: "semenovv",
	})
	redirect, err := service.Begin(context.Background(), TelegramBeginCommand{Mode: TelegramOIDCModeLogin})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Complete(context.Background(), TelegramCompleteCommand{State: telegramState(t, redirect), Code: "code"})
	if err != nil {
		t.Fatal(err)
	}
	if result.UserID != user.ID || len(users.users) != 1 {
		t.Fatalf("legacy identity did not preserve account: result=%+v users=%d", result, len(users.users))
	}
	if _, err := repo.FindIdentity(testTx{}, entity.AuthProviderTelegram, "42"); err != nil {
		t.Fatalf("canonical identity missing: %v", err)
	}
	if _, err := repo.FindIdentity(testTx{}, entity.AuthProviderTelegram, "oidc-subject"); !errors.Is(err, entity.ErrAuthIdentityNotFound) {
		t.Fatalf("legacy identity still present: %v", err)
	}
}

func TestTelegramOIDCRejectsSplitCanonicalAndLegacyAccounts(t *testing.T) {
	now := time.Date(2026, 8, 21, 3, 0, 0, 0, time.UTC)
	users := newFakeAuthRepo()
	repo := newFakeTelegramRepo()
	for _, item := range []struct {
		id      entity.AuthUserID
		email   string
		subject string
	}{{"established", "ishee@yandex.ru", "oidc-subject"}, {"duplicate", "telegram-x@telegram.invalid", "42"}} {
		user, err := entity.NewAuthUser(item.id, item.email, "password-hash", entity.AuthRoleUser, now)
		if err != nil {
			t.Fatal(err)
		}
		if err := users.Save(testTx{}, user); err != nil {
			t.Fatal(err)
		}
		if err := repo.SaveIdentity(testTx{}, &entity.AuthIdentity{
			Provider: entity.AuthProviderTelegram, Subject: item.subject, UserID: item.id,
			Username: "semenovv", CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	service := newTelegramService(repo, users, now, TelegramOIDCClaims{
		Subject: "42", LegacySubject: "oidc-subject", Username: "semenovv",
	})
	redirect, err := service.Begin(context.Background(), TelegramBeginCommand{Mode: TelegramOIDCModeLogin})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Complete(context.Background(), TelegramCompleteCommand{State: telegramState(t, redirect), Code: "code"})
	if !errors.Is(err, entity.ErrAuthIdentityLinked) {
		t.Fatalf("expected split identity conflict, got %v", err)
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
