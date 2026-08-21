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

type AuthIdentityRepository interface {
	SaveIdentity(tx Tx, identity *entity.AuthIdentity) error
	ReplaceIdentitySubject(tx Tx, provider entity.AuthProvider, oldSubject string, identity *entity.AuthIdentity) error
	FindIdentity(tx Tx, provider entity.AuthProvider, subject string) (*entity.AuthIdentity, error)
	ListIdentities(tx Tx, userID entity.AuthUserID) ([]entity.AuthIdentity, error)
	DeleteIdentity(tx Tx, userID entity.AuthUserID, provider entity.AuthProvider) error
}

type AuthOIDCFlowRepository interface {
	SaveOIDCFlow(tx Tx, flow *entity.AuthOIDCFlow) error
	ConsumeOIDCFlow(tx Tx, stateHash string, now time.Time) (*entity.AuthOIDCFlow, error)
}

type TelegramLoginChallengeRepository interface {
	SaveTelegramLoginChallenge(tx Tx, challenge *entity.TelegramLoginChallenge) error
	FindTelegramLoginChallenge(tx Tx, challengeHash string, browserBindingHash string, now time.Time) (*entity.TelegramLoginChallenge, error)
	FindTelegramLoginChallengeForBot(tx Tx, challengeHash string, now time.Time) (*entity.TelegramLoginChallenge, error)
	ClaimTelegramLoginChallengeActor(tx Tx, challengeHash string, subject string, now time.Time) (*entity.TelegramLoginChallenge, error)
	ApproveTelegramLoginChallenge(tx Tx, challengeHash string, subject string, username string, displayName string, now time.Time) (*entity.TelegramLoginChallenge, error)
	DenyTelegramLoginChallenge(tx Tx, challengeHash string, subject string, now time.Time) (*entity.TelegramLoginChallenge, error)
	LockTelegramLoginChallenge(tx Tx, challengeHash string, browserBindingHash string, now time.Time) (*entity.TelegramLoginChallenge, error)
	ConsumeTelegramLoginChallenge(tx Tx, challengeHash string, now time.Time) error
}

type UserPlayerLinkRepository interface {
	LinkPlayer(tx Tx, userID entity.AuthUserID, playerID entity.PlayerID) error
	UnlinkPlayer(tx Tx, userID entity.AuthUserID, playerID entity.PlayerID) error
	ListUserPlayers(tx Tx, userID entity.AuthUserID) ([]PlayerDTO, error)
	FindUserPlayer(tx Tx, userID entity.AuthUserID) (*PlayerDTO, error)
	IsPlayerLinked(tx Tx, playerID entity.PlayerID) (bool, error)
	IsPlayerLinkedToUser(tx Tx, userID entity.AuthUserID, playerID entity.PlayerID) (bool, error)
	ListUnlinkedPlayers(tx Tx, limit int, offset int) ([]AvailablePlayerDTO, error)
}

type AdminAccountOwnershipRepository interface {
	ListAccounts(tx Tx, query string, limit int, offset int) ([]AccountOwnershipDTO, int64, error)
	LockUser(tx Tx, userID entity.AuthUserID) error
	LockPlayer(tx Tx, playerID entity.PlayerID) error
	FindUserPlayer(tx Tx, userID entity.AuthUserID) (*PlayerDTO, error)
	FindPlayerOwner(tx Tx, playerID entity.PlayerID) (*entity.AuthUserID, error)
	LinkPlayer(tx Tx, userID entity.AuthUserID, playerID entity.PlayerID) error
	UnlinkPlayer(tx Tx, userID entity.AuthUserID, playerID entity.PlayerID) error
}
