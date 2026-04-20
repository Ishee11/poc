package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type AuthUserRepository interface {
	Save(tx Tx, user *entity.AuthUser) error
	FindUserByID(tx Tx, id entity.AuthUserID) (*entity.AuthUser, error)
	FindUserByEmail(tx Tx, email string) (*entity.AuthUser, error)
	UpdateLastLoginAt(tx Tx, id entity.AuthUserID, at time.Time) error
}

type AuthSessionRepository interface {
	SaveSession(tx Tx, session *entity.AuthSession) error
	FindSessionByTokenHash(tx Tx, tokenHash string) (*entity.AuthSession, error)
	TouchSession(tx Tx, id entity.AuthSessionID, lastSeenAt time.Time) error
	RevokeSession(tx Tx, id entity.AuthSessionID, revokedAt time.Time) error
	RevokeSessionByTokenHash(tx Tx, tokenHash string, revokedAt time.Time) error
}

type LoginAttemptRepository interface {
	SaveLoginAttempt(tx Tx, attempt *entity.LoginAttempt) error
	CountFailedLoginAttempts(tx Tx, email string, ip string, since time.Time) (int, error)
}

type UserPlayerLinkRepository interface {
	LinkPlayer(tx Tx, userID entity.AuthUserID, playerID entity.PlayerID) error
	UnlinkPlayer(tx Tx, userID entity.AuthUserID, playerID entity.PlayerID) error
	ListUserPlayers(tx Tx, userID entity.AuthUserID) ([]PlayerDTO, error)
	IsPlayerLinked(tx Tx, playerID entity.PlayerID) (bool, error)
	IsPlayerLinkedToUser(tx Tx, userID entity.AuthUserID, playerID entity.PlayerID) (bool, error)
	ListUnlinkedPlayers(tx Tx, limit int, offset int) ([]PlayerDTO, error)
}
