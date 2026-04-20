package infra

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/ishee11/poc/internal/entity"
)

const (
	argon2IDVersion     = 19
	argon2IDMemory      = 19 * 1024
	argon2IDIterations  = 2
	argon2IDParallelism = 1
	argon2IDSaltLength  = 16
	argon2IDKeyLength   = 32
)

var ErrInvalidPasswordHash = errors.New("invalid password hash")

type UUIDAuthUserIDGenerator struct{}

func (UUIDAuthUserIDGenerator) New() entity.AuthUserID {
	return entity.AuthUserID(uuid.NewString())
}

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

type Argon2IDPasswordHasher struct{}

func (Argon2IDPasswordHasher) HashPassword(password string) (string, error) {
	salt := make([]byte, argon2IDSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2IDIterations,
		argon2IDMemory,
		argon2IDParallelism,
		argon2IDKeyLength,
	)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2IDVersion,
		argon2IDMemory,
		argon2IDIterations,
		argon2IDParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func (h Argon2IDPasswordHasher) VerifyPassword(password string, passwordHash string) bool {
	params, salt, expectedHash, err := parseArgon2IDHash(passwordHash)
	if err != nil {
		return false
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		uint32(len(expectedHash)),
	)

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}

type argon2IDParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func parseArgon2IDHash(encoded string) (argon2IDParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return argon2IDParams{}, nil, nil, ErrInvalidPasswordHash
	}

	params, err := parseArgon2IDParams(parts[3])
	if err != nil {
		return argon2IDParams{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2IDParams{}, nil, nil, ErrInvalidPasswordHash
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2IDParams{}, nil, nil, ErrInvalidPasswordHash
	}

	return params, salt, hash, nil
}

func parseArgon2IDParams(encoded string) (argon2IDParams, error) {
	parts := strings.Split(encoded, ",")
	if len(parts) != 3 {
		return argon2IDParams{}, ErrInvalidPasswordHash
	}

	values := make(map[string]string, len(parts))
	for _, part := range parts {
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) != 2 {
			return argon2IDParams{}, ErrInvalidPasswordHash
		}
		values[keyValue[0]] = keyValue[1]
	}

	memory, err := strconv.ParseUint(values["m"], 10, 32)
	if err != nil {
		return argon2IDParams{}, ErrInvalidPasswordHash
	}

	iterations, err := strconv.ParseUint(values["t"], 10, 32)
	if err != nil {
		return argon2IDParams{}, ErrInvalidPasswordHash
	}

	parallelism, err := strconv.ParseUint(values["p"], 10, 8)
	if err != nil {
		return argon2IDParams{}, ErrInvalidPasswordHash
	}

	return argon2IDParams{
		memory:      uint32(memory),
		iterations:  uint32(iterations),
		parallelism: uint8(parallelism),
	}, nil
}
