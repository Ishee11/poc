package entity

import (
	"errors"
	"testing"
	"time"
)

func TestNewAuthUser(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

	user, err := NewAuthUser("user-1", "admin@example.com", "hash", AuthRoleAdmin, now)
	if err != nil {
		t.Fatalf("NewAuthUser returned error: %v", err)
	}

	if user.ID != "user-1" {
		t.Fatalf("unexpected id: %s", user.ID)
	}
	if user.Status != AuthUserStatusActive {
		t.Fatalf("unexpected status: %s", user.Status)
	}
	if !user.CreatedAt.Equal(now) || !user.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected timestamps: created=%s updated=%s", user.CreatedAt, user.UpdatedAt)
	}
}

func TestNewAuthUserRejectsInvalidRole(t *testing.T) {
	_, err := NewAuthUser("user-1", "admin@example.com", "hash", AuthRole("owner"), time.Now())
	if !errors.Is(err, ErrInvalidAuthRole) {
		t.Fatalf("expected ErrInvalidAuthRole, got %v", err)
	}
}

func TestRestoreAuthUserRejectsInvalidStatus(t *testing.T) {
	now := time.Now()

	_, err := RestoreAuthUser(
		"user-1",
		"admin@example.com",
		"hash",
		AuthRoleAdmin,
		AuthUserStatus("blocked"),
		now,
		now,
		nil,
	)
	if !errors.Is(err, ErrInvalidAuthStatus) {
		t.Fatalf("expected ErrInvalidAuthStatus, got %v", err)
	}
}

func TestAuthSessionState(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	session := NewAuthSession(
		"session-1",
		"user-1",
		"token-hash",
		"agent",
		"127.0.0.1",
		now,
		now.Add(time.Hour),
	)

	if session.Expired(now) {
		t.Fatal("session should not be expired before expires_at")
	}
	if !session.Expired(now.Add(time.Hour)) {
		t.Fatal("session should be expired at expires_at")
	}
	if session.Revoked() {
		t.Fatal("new session should not be revoked")
	}

	revokedAt := now.Add(time.Minute)
	session.RevokedAt = &revokedAt

	if !session.Revoked() {
		t.Fatal("session should be revoked when revoked_at is set")
	}
}
