package infra

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/google/uuid"

	"github.com/ishee11/poc/internal/entity"
)

type UUIDAuthSessionIDGenerator struct{}

func (UUIDAuthSessionIDGenerator) New() entity.AuthSessionID {
	return entity.AuthSessionID(uuid.NewString())
}

type UUIDLoginAttemptIDGenerator struct{}

func (UUIDLoginAttemptIDGenerator) New() entity.LoginAttemptID {
	return entity.LoginAttemptID(uuid.NewString())
}

type SecureTokenGenerator struct {
	Bytes int
}

func (g SecureTokenGenerator) NewToken() (string, error) {
	size := g.Bytes
	if size == 0 {
		size = 32
	}

	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

type SHA256TokenHasher struct{}

func (SHA256TokenHasher) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
