package usecase

import (
	"errors"
	"strings"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

const authSessionTouchInterval = time.Minute

type AuthSessionIDGenerator interface {
	New() entity.AuthSessionID
}

type AuthUserIDGenerator interface {
	New() entity.AuthUserID
}

type LoginAttemptIDGenerator interface {
	New() entity.LoginAttemptID
}

type TokenGenerator interface {
	NewToken() (string, error)
}

type TokenHasher interface {
	HashToken(token string) string
}

type PasswordVerifier interface {
	VerifyPassword(password string, passwordHash string) bool
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type AuthPolicy struct {
	SessionTTL        time.Duration
	IdleTTL           time.Duration
	RateLimitWindow   time.Duration
	MaxFailedAttempts int
}

func DefaultAuthPolicy() AuthPolicy {
	return AuthPolicy{
		SessionTTL:        12 * time.Hour,
		IdleTTL:           2 * time.Hour,
		RateLimitWindow:   time.Minute,
		MaxFailedAttempts: 5,
	}
}

type AuthPrincipal struct {
	UserID    entity.AuthUserID
	Email     string
	Role      entity.AuthRole
	SessionID entity.AuthSessionID
}

type LoginCommand struct {
	Email     string
	Password  string
	UserAgent string
	IP        string
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      AuthPrincipal
}

type AuthService struct {
	userRepo     AuthUserRepository
	sessionRepo  AuthSessionRepository
	attemptRepo  LoginAttemptRepository
	txManager    TxManager
	sessionIDGen AuthSessionIDGenerator
	attemptIDGen LoginAttemptIDGenerator
	tokenGen     TokenGenerator
	tokenHasher  TokenHasher
	passwords    PasswordVerifier
	clock        Clock
	policy       AuthPolicy
}

func NewAuthService(
	userRepo AuthUserRepository,
	sessionRepo AuthSessionRepository,
	attemptRepo LoginAttemptRepository,
	txManager TxManager,
	sessionIDGen AuthSessionIDGenerator,
	attemptIDGen LoginAttemptIDGenerator,
	tokenGen TokenGenerator,
	tokenHasher TokenHasher,
	passwords PasswordVerifier,
	clock Clock,
	policy AuthPolicy,
) *AuthService {
	if clock == nil {
		clock = SystemClock{}
	}
	if policy.SessionTTL == 0 {
		policy.SessionTTL = DefaultAuthPolicy().SessionTTL
	}
	if policy.IdleTTL == 0 {
		policy.IdleTTL = DefaultAuthPolicy().IdleTTL
	}
	if policy.RateLimitWindow == 0 {
		policy.RateLimitWindow = DefaultAuthPolicy().RateLimitWindow
	}
	if policy.MaxFailedAttempts == 0 {
		policy.MaxFailedAttempts = DefaultAuthPolicy().MaxFailedAttempts
	}

	return &AuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		attemptRepo:  attemptRepo,
		txManager:    txManager,
		sessionIDGen: sessionIDGen,
		attemptIDGen: attemptIDGen,
		tokenGen:     tokenGen,
		tokenHasher:  tokenHasher,
		passwords:    passwords,
		clock:        clock,
		policy:       policy,
	}
}

func (s *AuthService) Login(cmd LoginCommand) (*LoginResult, error) {
	var result *LoginResult
	var authErr error
	email := strings.TrimSpace(cmd.Email)

	err := s.txManager.RunInTx(func(tx Tx) error {
		now := s.clock.Now()

		failed, err := s.attemptRepo.CountFailedLoginAttempts(
			tx,
			email,
			cmd.IP,
			now.Add(-s.policy.RateLimitWindow),
		)
		if err != nil {
			return err
		}
		if failed >= s.policy.MaxFailedAttempts {
			return entity.ErrAuthRateLimited
		}

		user, err := s.userRepo.FindUserByEmail(tx, email)
		if err != nil {
			if errors.Is(err, entity.ErrAuthUserNotFound) {
				authErr = entity.ErrInvalidCredentials
				return s.recordLoginAttempt(tx, email, cmd.IP, false, now)
			}
			return err
		}

		if user.Status != entity.AuthUserStatusActive ||
			!s.passwords.VerifyPassword(cmd.Password, user.PasswordHash) {
			authErr = entity.ErrInvalidCredentials
			return s.recordLoginAttempt(tx, email, cmd.IP, false, now)
		}

		token, err := s.tokenGen.NewToken()
		if err != nil {
			return err
		}

		session := entity.NewAuthSession(
			s.sessionIDGen.New(),
			user.ID,
			s.tokenHasher.HashToken(token),
			cmd.UserAgent,
			cmd.IP,
			now,
			now.Add(s.policy.SessionTTL),
		)

		if err := s.sessionRepo.SaveSession(tx, session); err != nil {
			return err
		}
		if err := s.userRepo.UpdateLastLoginAt(tx, user.ID, now); err != nil {
			return err
		}
		if err := s.recordLoginAttempt(tx, email, cmd.IP, true, now); err != nil {
			return err
		}

		result = &LoginResult{
			Token:     token,
			ExpiresAt: session.ExpiresAt,
			User: AuthPrincipal{
				UserID:    user.ID,
				Email:     user.Email,
				Role:      user.Role,
				SessionID: session.ID,
			},
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if authErr != nil {
		return nil, authErr
	}

	return result, nil
}

func (s *AuthService) CurrentUser(rawToken string) (*AuthPrincipal, error) {
	if rawToken == "" {
		return nil, entity.ErrUnauthorized
	}

	var principal *AuthPrincipal
	err := s.txManager.RunInTx(func(tx Tx) error {
		now := s.clock.Now()

		session, err := s.sessionRepo.FindSessionByTokenHash(tx, s.tokenHasher.HashToken(rawToken))
		if err != nil {
			if errors.Is(err, entity.ErrAuthSessionNotFound) {
				return entity.ErrUnauthorized
			}
			return err
		}

		if session.Revoked() || session.Expired(now) || now.Sub(session.LastSeenAt) > s.policy.IdleTTL {
			_ = s.sessionRepo.RevokeSession(tx, session.ID, now)
			return entity.ErrUnauthorized
		}

		user, err := s.userRepo.FindUserByID(tx, session.UserID)
		if err != nil {
			if errors.Is(err, entity.ErrAuthUserNotFound) {
				return entity.ErrUnauthorized
			}
			return err
		}
		if user.Status != entity.AuthUserStatusActive {
			return entity.ErrUnauthorized
		}

		if now.Sub(session.LastSeenAt) >= authSessionTouchInterval {
			if err := s.sessionRepo.TouchSession(tx, session.ID, now); err != nil {
				return err
			}
		}

		principal = &AuthPrincipal{
			UserID:    user.ID,
			Email:     user.Email,
			Role:      user.Role,
			SessionID: session.ID,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return principal, nil
}

func (s *AuthService) Logout(rawToken string) error {
	if rawToken == "" {
		return nil
	}

	return s.txManager.RunInTx(func(tx Tx) error {
		err := s.sessionRepo.RevokeSessionByTokenHash(
			tx,
			s.tokenHasher.HashToken(rawToken),
			s.clock.Now(),
		)
		if errors.Is(err, entity.ErrAuthSessionNotFound) {
			return nil
		}
		return err
	})
}

func (s *AuthService) RequireRole(principal AuthPrincipal, allowed ...entity.AuthRole) error {
	for _, role := range allowed {
		if principal.Role == role {
			return nil
		}
	}
	return entity.ErrForbidden
}

func (s *AuthService) recordLoginAttempt(tx Tx, email string, ip string, success bool, now time.Time) error {
	return s.attemptRepo.SaveLoginAttempt(
		tx,
		entity.NewLoginAttempt(s.attemptIDGen.New(), email, ip, success, now),
	)
}
