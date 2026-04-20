package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time {
	return c.now
}

type fakeAuthSessionIDGen struct{ next entity.AuthSessionID }

func (g fakeAuthSessionIDGen) New() entity.AuthSessionID {
	if g.next == "" {
		return "auth-session-1"
	}
	return g.next
}

type fakeAuthUserIDGen struct{ next entity.AuthUserID }

func (g fakeAuthUserIDGen) New() entity.AuthUserID {
	if g.next == "" {
		return "auth-user-1"
	}
	return g.next
}

type fakeLoginAttemptIDGen struct {
	next entity.LoginAttemptID
	n    int
}

func (g *fakeLoginAttemptIDGen) New() entity.LoginAttemptID {
	if g.next != "" && g.n == 0 {
		g.n++
		return g.next
	}
	g.n++
	return entity.LoginAttemptID("login-attempt-" + string(rune('0'+g.n)))
}

type fakeTokenGenerator struct {
	token string
	err   error
}

func (g fakeTokenGenerator) NewToken() (string, error) {
	if g.err != nil {
		return "", g.err
	}
	if g.token == "" {
		return "raw-token", nil
	}
	return g.token, nil
}

type fakeTokenHasher struct{}

func (fakeTokenHasher) HashToken(token string) string {
	return "hash:" + token
}

type fakePasswordVerifier struct{ ok bool }

func (v fakePasswordVerifier) VerifyPassword(_, _ string) bool {
	return v.ok
}

type fakePasswordHasher struct {
	hash string
	err  error
}

func (h fakePasswordHasher) HashPassword(_ string) (string, error) {
	if h.err != nil {
		return "", h.err
	}
	if h.hash == "" {
		return "password-hash", nil
	}
	return h.hash, nil
}

type fakeAuthRepo struct {
	users       map[entity.AuthUserID]*entity.AuthUser
	usersEmail  map[string]entity.AuthUserID
	sessions    map[entity.AuthSessionID]*entity.AuthSession
	tokenIndex  map[string]entity.AuthSessionID
	attempts    []*entity.LoginAttempt
	failedCount int
}

func newFakeAuthRepo() *fakeAuthRepo {
	return &fakeAuthRepo{
		users:      make(map[entity.AuthUserID]*entity.AuthUser),
		usersEmail: make(map[string]entity.AuthUserID),
		sessions:   make(map[entity.AuthSessionID]*entity.AuthSession),
		tokenIndex: make(map[string]entity.AuthSessionID),
	}
}

func (r *fakeAuthRepo) Save(_ Tx, user *entity.AuthUser) error {
	r.users[user.ID] = user
	r.usersEmail[user.Email] = user.ID
	return nil
}

func (r *fakeAuthRepo) FindUserByID(_ Tx, id entity.AuthUserID) (*entity.AuthUser, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, entity.ErrAuthUserNotFound
	}
	return user, nil
}

func (r *fakeAuthRepo) FindUserByEmail(_ Tx, email string) (*entity.AuthUser, error) {
	id, ok := r.usersEmail[email]
	if !ok {
		return nil, entity.ErrAuthUserNotFound
	}
	return r.users[id], nil
}

func (r *fakeAuthRepo) UpdateLastLoginAt(_ Tx, id entity.AuthUserID, at time.Time) error {
	user, ok := r.users[id]
	if !ok {
		return entity.ErrAuthUserNotFound
	}
	user.LastLoginAt = &at
	return nil
}

func (r *fakeAuthRepo) SaveSession(_ Tx, session *entity.AuthSession) error {
	r.sessions[session.ID] = session
	r.tokenIndex[session.TokenHash] = session.ID
	return nil
}

func (r *fakeAuthRepo) FindSessionByTokenHash(_ Tx, tokenHash string) (*entity.AuthSession, error) {
	id, ok := r.tokenIndex[tokenHash]
	if !ok {
		return nil, entity.ErrAuthSessionNotFound
	}
	session := r.sessions[id]
	if session.Revoked() {
		return nil, entity.ErrAuthSessionNotFound
	}
	return session, nil
}

func (r *fakeAuthRepo) TouchSession(_ Tx, id entity.AuthSessionID, lastSeenAt time.Time) error {
	session, ok := r.sessions[id]
	if !ok || session.Revoked() {
		return entity.ErrAuthSessionNotFound
	}
	session.LastSeenAt = lastSeenAt
	return nil
}

func (r *fakeAuthRepo) RevokeSession(_ Tx, id entity.AuthSessionID, revokedAt time.Time) error {
	session, ok := r.sessions[id]
	if !ok || session.Revoked() {
		return entity.ErrAuthSessionNotFound
	}
	session.RevokedAt = &revokedAt
	return nil
}

func (r *fakeAuthRepo) RevokeSessionByTokenHash(_ Tx, tokenHash string, revokedAt time.Time) error {
	id, ok := r.tokenIndex[tokenHash]
	if !ok {
		return entity.ErrAuthSessionNotFound
	}
	return r.RevokeSession(testTx{}, id, revokedAt)
}

func (r *fakeAuthRepo) SaveLoginAttempt(_ Tx, attempt *entity.LoginAttempt) error {
	r.attempts = append(r.attempts, attempt)
	return nil
}

func (r *fakeAuthRepo) CountFailedLoginAttempts(_ Tx, _ string, _ string, _ time.Time) (int, error) {
	return r.failedCount, nil
}

func newTestAuthService(repo *fakeAuthRepo, now time.Time, passwords PasswordVerifier) *AuthService {
	return NewAuthService(
		repo,
		repo,
		repo,
		fakeTxManager{},
		fakeAuthSessionIDGen{next: "session-1"},
		&fakeLoginAttemptIDGen{next: "attempt-1"},
		fakeTokenGenerator{token: "token-1"},
		fakeTokenHasher{},
		passwords,
		fakeClock{now: now},
		AuthPolicy{
			SessionTTL:        time.Hour,
			IdleTTL:           30 * time.Minute,
			RateLimitWindow:   time.Minute,
			MaxFailedAttempts: 5,
		},
	)
}

func TestAuthServiceLoginSuccess(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	user, err := entity.NewAuthUser("user-1", "admin@example.com", "hash", entity.AuthRoleAdmin, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(testTx{}, user); err != nil {
		t.Fatal(err)
	}

	service := newTestAuthService(repo, now, fakePasswordVerifier{ok: true})
	result, err := service.Login(LoginCommand{
		Email:     "admin@example.com",
		Password:  "password",
		UserAgent: "agent",
		IP:        "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if result.Token != "token-1" {
		t.Fatalf("unexpected token: %s", result.Token)
	}
	if result.User.UserID != "user-1" || result.User.Role != entity.AuthRoleAdmin {
		t.Fatalf("unexpected principal: %+v", result.User)
	}
	if len(repo.sessions) != 1 {
		t.Fatalf("expected one session, got %d", len(repo.sessions))
	}
	if len(repo.attempts) != 1 || !repo.attempts[0].Success {
		t.Fatalf("expected successful login attempt, got %+v", repo.attempts)
	}
	if user.LastLoginAt == nil || !user.LastLoginAt.Equal(now) {
		t.Fatalf("last login not updated: %v", user.LastLoginAt)
	}
}

func TestAuthServiceLoginInvalidCredentialsRecordsAttempt(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()

	service := newTestAuthService(repo, now, fakePasswordVerifier{ok: false})
	_, err := service.Login(LoginCommand{
		Email:    "missing@example.com",
		Password: "wrong",
		IP:       "127.0.0.1",
	})

	if !errors.Is(err, entity.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if len(repo.attempts) != 1 || repo.attempts[0].Success {
		t.Fatalf("expected failed login attempt, got %+v", repo.attempts)
	}
}

func TestAuthServiceCurrentUserTouchesSession(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	user, err := entity.NewAuthUser("user-1", "admin@example.com", "hash", entity.AuthRoleMember, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(testTx{}, user); err != nil {
		t.Fatal(err)
	}
	session := entity.NewAuthSession(
		"session-1",
		user.ID,
		"hash:token-1",
		"agent",
		"127.0.0.1",
		now.Add(-10*time.Minute),
		now.Add(time.Hour),
	)
	if err := repo.SaveSession(testTx{}, session); err != nil {
		t.Fatal(err)
	}

	service := newTestAuthService(repo, now, fakePasswordVerifier{ok: true})
	principal, err := service.CurrentUser("token-1")
	if err != nil {
		t.Fatalf("CurrentUser returned error: %v", err)
	}

	if principal.UserID != user.ID {
		t.Fatalf("unexpected principal: %+v", principal)
	}
	if !session.LastSeenAt.Equal(now) {
		t.Fatalf("session was not touched: %s", session.LastSeenAt)
	}
}

func TestAuthServiceCurrentUserRevokesExpiredSession(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	user, err := entity.NewAuthUser("user-1", "admin@example.com", "hash", entity.AuthRoleMember, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(testTx{}, user); err != nil {
		t.Fatal(err)
	}
	session := entity.NewAuthSession(
		"session-1",
		user.ID,
		"hash:token-1",
		"agent",
		"127.0.0.1",
		now.Add(-2*time.Hour),
		now.Add(-time.Hour),
	)
	if err := repo.SaveSession(testTx{}, session); err != nil {
		t.Fatal(err)
	}

	service := newTestAuthService(repo, now, fakePasswordVerifier{ok: true})
	_, err = service.CurrentUser("token-1")

	if !errors.Is(err, entity.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if !session.Revoked() {
		t.Fatal("expired session should be revoked")
	}
}

func TestAuthServiceRequireRole(t *testing.T) {
	service := newTestAuthService(newFakeAuthRepo(), time.Now(), fakePasswordVerifier{ok: true})

	err := service.RequireRole(AuthPrincipal{Role: entity.AuthRoleViewer}, entity.AuthRoleAdmin, entity.AuthRoleMember)
	if !errors.Is(err, entity.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	err = service.RequireRole(AuthPrincipal{Role: entity.AuthRoleMember}, entity.AuthRoleAdmin, entity.AuthRoleMember)
	if err != nil {
		t.Fatalf("expected allowed role, got %v", err)
	}
}

func TestSeedAdminUseCaseCreatesAdmin(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	uc := NewSeedAdminUseCase(
		repo,
		fakeTxManager{},
		fakeAuthUserIDGen{next: "admin-1"},
		fakePasswordHasher{hash: "hash"},
		fakeClock{now: now},
	)

	err := uc.Execute(SeedAdminCommand{
		Email:    " admin@example.com ",
		Password: "long-password",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	user, err := repo.FindUserByEmail(testTx{}, "admin@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "admin-1" || user.Role != entity.AuthRoleAdmin || user.PasswordHash != "hash" {
		t.Fatalf("unexpected seeded admin: %+v", user)
	}
}

func TestSeedAdminUseCaseIsNoopWhenAdminExists(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	existing, err := entity.NewAuthUser("admin-1", "admin@example.com", "old-hash", entity.AuthRoleAdmin, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(testTx{}, existing); err != nil {
		t.Fatal(err)
	}

	uc := NewSeedAdminUseCase(
		repo,
		fakeTxManager{},
		fakeAuthUserIDGen{next: "admin-2"},
		fakePasswordHasher{hash: "new-hash"},
		fakeClock{now: now},
	)

	err = uc.Execute(SeedAdminCommand{
		Email:    "admin@example.com",
		Password: "long-password",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	user, err := repo.FindUserByEmail(testTx{}, "admin@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "admin-1" || user.PasswordHash != "old-hash" {
		t.Fatalf("existing admin should not be overwritten: %+v", user)
	}
}

func TestSeedAdminUseCaseRejectsShortPassword(t *testing.T) {
	uc := NewSeedAdminUseCase(
		newFakeAuthRepo(),
		fakeTxManager{},
		fakeAuthUserIDGen{},
		fakePasswordHasher{},
		fakeClock{now: time.Now()},
	)

	err := uc.Execute(SeedAdminCommand{
		Email:    "admin@example.com",
		Password: "short",
	})
	if !errors.Is(err, entity.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestSeedAdminUseCaseRejectsMissingEmail(t *testing.T) {
	uc := NewSeedAdminUseCase(
		newFakeAuthRepo(),
		fakeTxManager{},
		fakeAuthUserIDGen{},
		fakePasswordHasher{},
		fakeClock{now: time.Now()},
	)

	err := uc.Execute(SeedAdminCommand{
		Password: "long-password",
	})
	if !errors.Is(err, entity.ErrInvalidAuthEmail) {
		t.Fatalf("expected ErrInvalidAuthEmail, got %v", err)
	}
}
