package app

import (
	"bytes"
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/mail"
	"net/url"
	"strings"
)

func validatePushConfig(cfg *PushConfig) error {
	cfg.Subject = normalizePushSubject(cfg.Subject)
	cfg.PublicKey = strings.TrimSpace(cfg.PublicKey)
	cfg.PrivateKey = strings.TrimSpace(cfg.PrivateKey)

	if !cfg.Enabled {
		return nil
	}

	if cfg.Subject == "" || cfg.PublicKey == "" || cfg.PrivateKey == "" {
		return fmt.Errorf("PUSH_ENABLED=true requires PUSH_SUBJECT, PUSH_VAPID_PUBLIC_KEY and PUSH_VAPID_PRIVATE_KEY")
	}
	if err := validatePushSubject(cfg.Subject); err != nil {
		return fmt.Errorf("PUSH_SUBJECT is invalid: %w", err)
	}

	publicKey, err := decodeVAPIDKey(cfg.PublicKey)
	if err != nil {
		return fmt.Errorf("PUSH_VAPID_PUBLIC_KEY is invalid: %w", err)
	}
	privateKey, err := decodeVAPIDKey(cfg.PrivateKey)
	if err != nil {
		return fmt.Errorf("PUSH_VAPID_PRIVATE_KEY is invalid: %w", err)
	}
	if err := validateVAPIDKeyPair(publicKey, privateKey); err != nil {
		return err
	}

	cfg.PublicKey = base64.RawURLEncoding.EncodeToString(publicKey)
	cfg.PrivateKey = base64.RawURLEncoding.EncodeToString(privateKey)
	return nil
}

func validatePushSubject(subject string) error {
	lower := strings.ToLower(subject)
	if strings.HasPrefix(lower, "https://") {
		parsed, err := url.Parse(subject)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("must be an HTTPS URL or email address")
		}
		return nil
	}
	if strings.HasPrefix(lower, "http://") || strings.Contains(subject, "://") {
		return fmt.Errorf("must be an HTTPS URL or email address")
	}

	address, err := mail.ParseAddress(subject)
	if err != nil || address.Address != subject {
		return fmt.Errorf("must be an HTTPS URL or email address")
	}
	return nil
}

func decodeVAPIDKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("empty key")
	}

	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}

	return nil, fmt.Errorf("must be base64url encoded")
}

func validateVAPIDKeyPair(publicKey, privateKey []byte) error {
	curve := elliptic.P256()
	if len(privateKey) != 32 {
		return fmt.Errorf("PUSH_VAPID_PRIVATE_KEY is invalid: expected 32-byte P-256 key")
	}
	privateScalar := new(big.Int).SetBytes(privateKey)
	if privateScalar.Sign() <= 0 || privateScalar.Cmp(curve.Params().N) >= 0 {
		return fmt.Errorf("PUSH_VAPID_PRIVATE_KEY is invalid: scalar is out of range")
	}

	if len(publicKey) != 65 || publicKey[0] != 4 {
		return fmt.Errorf("PUSH_VAPID_PUBLIC_KEY is invalid: expected uncompressed 65-byte P-256 key")
	}
	x, y := elliptic.Unmarshal(curve, publicKey)
	if x == nil || y == nil {
		return fmt.Errorf("PUSH_VAPID_PUBLIC_KEY is invalid: point is not on P-256")
	}

	derivedX, derivedY := curve.ScalarBaseMult(privateKey)
	derivedPublicKey := elliptic.Marshal(curve, derivedX, derivedY)
	if !bytes.Equal(derivedPublicKey, publicKey) {
		return fmt.Errorf("PUSH_VAPID_PUBLIC_KEY does not match PUSH_VAPID_PRIVATE_KEY")
	}

	return nil
}
