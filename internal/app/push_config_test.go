package app

import (
	"strings"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func TestValidatePushConfigNormalizesMailtoSubject(t *testing.T) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate VAPID keys: %v", err)
	}

	cfg := PushConfig{
		Enabled:    true,
		Subject:    " mailto:admin@example.com ",
		PublicKey:  " " + publicKey + " ",
		PrivateKey: " " + privateKey + " ",
	}

	if err := validatePushConfig(&cfg); err != nil {
		t.Fatalf("validate push config: %v", err)
	}
	if cfg.Subject != "admin@example.com" {
		t.Fatalf("subject = %q, want %q", cfg.Subject, "admin@example.com")
	}
	if strings.Contains(cfg.Subject, "mailto:") {
		t.Fatalf("subject still contains mailto prefix: %q", cfg.Subject)
	}
	if cfg.PublicKey != publicKey {
		t.Fatalf("public key was not normalized as expected")
	}
	if cfg.PrivateKey != privateKey {
		t.Fatalf("private key was not normalized as expected")
	}
}

func TestValidatePushConfigAllowsHTTPSSubject(t *testing.T) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate VAPID keys: %v", err)
	}

	cfg := PushConfig{
		Enabled:    true,
		Subject:    "https://poker.example.com/push-contact",
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	if err := validatePushConfig(&cfg); err != nil {
		t.Fatalf("validate push config: %v", err)
	}
}

func TestValidatePushConfigRejectsMismatchedVAPIDKeys(t *testing.T) {
	privateKey, _, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate first VAPID keys: %v", err)
	}
	_, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate second VAPID keys: %v", err)
	}

	cfg := PushConfig{
		Enabled:    true,
		Subject:    "admin@example.com",
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	err = validatePushConfig(&cfg)
	if err == nil {
		t.Fatalf("validate push config unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("validate push config error = %v, want key mismatch", err)
	}
}

func TestValidatePushConfigRejectsHTTPSubject(t *testing.T) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate VAPID keys: %v", err)
	}

	cfg := PushConfig{
		Enabled:    true,
		Subject:    "http://poker.example.com",
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	if err := validatePushConfig(&cfg); err == nil {
		t.Fatalf("validate push config unexpectedly succeeded")
	}
}
