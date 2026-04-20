package entity

import (
	"errors"
	"time"
)

type AuthUserID string
type AuthSessionID string
type LoginAttemptID string

type AuthRole string

const (
	AuthRoleAdmin  AuthRole = "admin"
	AuthRoleMember AuthRole = "member"
	AuthRoleViewer AuthRole = "viewer"
)

func (r AuthRole) Valid() bool {
	switch r {
	case AuthRoleAdmin, AuthRoleMember, AuthRoleViewer:
		return true
	default:
		return false
	}
}

type AuthUserStatus string

const (
	AuthUserStatusActive   AuthUserStatus = "active"
	AuthUserStatusDisabled AuthUserStatus = "disabled"
)

func (s AuthUserStatus) Valid() bool {
	switch s {
	case AuthUserStatusActive, AuthUserStatusDisabled:
		return true
	default:
		return false
	}
}

var (
	ErrAuthUserNotFound    = errors.New("auth user not found")
	ErrAuthSessionNotFound = errors.New("auth session not found")
	ErrInvalidAuthRole     = errors.New("invalid auth role")
	ErrInvalidAuthStatus   = errors.New("invalid auth status")
)

type AuthUser struct {
	ID           AuthUserID
	Email        string
	PasswordHash string
	Role         AuthRole
	Status       AuthUserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

func NewAuthUser(
	id AuthUserID,
	email string,
	passwordHash string,
	role AuthRole,
	now time.Time,
) (*AuthUser, error) {
	if !role.Valid() {
		return nil, ErrInvalidAuthRole
	}

	return &AuthUser{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Status:       AuthUserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func RestoreAuthUser(
	id AuthUserID,
	email string,
	passwordHash string,
	role AuthRole,
	status AuthUserStatus,
	createdAt time.Time,
	updatedAt time.Time,
	lastLoginAt *time.Time,
) (*AuthUser, error) {
	if !role.Valid() {
		return nil, ErrInvalidAuthRole
	}
	if !status.Valid() {
		return nil, ErrInvalidAuthStatus
	}

	return &AuthUser{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Status:       status,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		LastLoginAt:  lastLoginAt,
	}, nil
}

type AuthSession struct {
	ID         AuthSessionID
	UserID     AuthUserID
	TokenHash  string
	UserAgent  string
	IP         string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

func NewAuthSession(
	id AuthSessionID,
	userID AuthUserID,
	tokenHash string,
	userAgent string,
	ip string,
	now time.Time,
	expiresAt time.Time,
) *AuthSession {
	return &AuthSession{
		ID:         id,
		UserID:     userID,
		TokenHash:  tokenHash,
		UserAgent:  userAgent,
		IP:         ip,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  expiresAt,
	}
}

func (s *AuthSession) Expired(now time.Time) bool {
	return !s.ExpiresAt.After(now)
}

func (s *AuthSession) Revoked() bool {
	return s.RevokedAt != nil
}

type LoginAttempt struct {
	ID        LoginAttemptID
	Email     string
	IP        string
	Success   bool
	CreatedAt time.Time
}

func NewLoginAttempt(id LoginAttemptID, email string, ip string, success bool, now time.Time) *LoginAttempt {
	return &LoginAttempt{
		ID:        id,
		Email:     email,
		IP:        ip,
		Success:   success,
		CreatedAt: now,
	}
}
