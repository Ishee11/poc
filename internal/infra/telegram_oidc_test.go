package infra

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func encodeJWTPart(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func signedTelegramToken(t *testing.T, key *rsa.PrivateKey, nonce string, audience string) string {
	t.Helper()
	header := encodeJWTPart(t, map[string]any{"alg": "RS256", "kid": "test-key", "typ": "JWT"})
	claims := encodeJWTPart(t, map[string]any{
		"iss": "https://oauth.telegram.org", "aud": audience, "sub": "telegram-42",
		"id":  int64(42),
		"exp": time.Now().Add(time.Minute).Unix(), "nonce": nonce,
		"name": "Telegram User", "preferred_username": "telegram_user",
	})
	unsigned := header + "." + claims
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestTelegramOIDCClientExchangesAndValidatesRS256IDToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	exponent := big.NewInt(int64(key.PublicKey.E)).Bytes()
	client := NewTelegramOIDCClient(TelegramOIDCClientConfig{
		ClientID: "123", ClientSecret: "secret",
		TokenURL: "https://telegram.test/token", JWKSURL: "https://telegram.test/jwks",
	})
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		status := http.StatusOK
		switch r.URL.Path {
		case "/token":
			username, password, ok := r.BasicAuth()
			if !ok || username != "123" || password != "secret" {
				t.Fatalf("unexpected basic auth: %q %q %v", username, password, ok)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("code_verifier") != "verifier" || r.Form.Get("code") != "code" {
				t.Fatalf("unexpected token form: %v", r.Form)
			}
			body = `{"id_token":"` + signedTelegramToken(t, key, "nonce", "123") + `"}`
		case "/jwks":
			raw, marshalErr := json.Marshal(map[string]any{"keys": []map[string]string{{
				"kid": "test-key", "kty": "RSA", "use": "sig",
				"n": base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString(exponent),
			}}})
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			body = string(raw)
		default:
			status = http.StatusNotFound
		}
		return &http.Response{
			StatusCode: status,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})}
	claims, err := client.Exchange(context.Background(), "code", "verifier", "https://poc.test/callback", "nonce")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "42" || claims.LegacySubject != "telegram-42" || claims.Username != "telegram_user" || claims.DisplayName != "Telegram User" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTelegramOIDCClientRejectsWrongNonce(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	client := NewTelegramOIDCClient(TelegramOIDCClientConfig{ClientID: "123"})
	client.keys["test-key"] = &key.PublicKey
	client.keysAt = time.Now()
	_, err = client.validateIDToken(context.Background(), signedTelegramToken(t, key, "actual", "123"), "expected")
	if err == nil {
		t.Fatal("expected nonce validation error")
	}
}

func TestTelegramOIDCClientRejectsTokenWithoutProfileUserID(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	client := NewTelegramOIDCClient(TelegramOIDCClientConfig{ClientID: "123"})
	client.keys["test-key"] = &key.PublicKey
	client.keysAt = time.Now()
	token := signedTelegramToken(t, key, "nonce", "123")
	parts := strings.Split(token, ".")
	var claims map[string]any
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(raw, &claims) != nil {
		t.Fatal("decode signed test token")
	}
	delete(claims, "id")
	unsigned := parts[0] + "." + encodeJWTPart(t, claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.validateIDToken(context.Background(), unsigned+"."+base64.RawURLEncoding.EncodeToString(signature), "nonce")
	if err == nil {
		t.Fatal("expected missing Telegram profile id to be rejected")
	}
}

func TestTelegramOIDCClientClassifiesProviderTransportFailure(t *testing.T) {
	client := NewTelegramOIDCClient(TelegramOIDCClientConfig{
		ClientID: "123", ClientSecret: "secret", TokenURL: "https://telegram.test/token",
	})
	client.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})}

	_, err := client.Exchange(context.Background(), "code", "verifier", "https://poc.test/callback", "nonce")
	if !errors.Is(err, entity.ErrTelegramProviderUnavailable) {
		t.Fatalf("expected provider unavailable, got %v", err)
	}
}

func TestTelegramOIDCClientClassifiesProviderServiceFailure(t *testing.T) {
	client := NewTelegramOIDCClient(TelegramOIDCClientConfig{
		ClientID: "123", ClientSecret: "secret", TokenURL: "https://telegram.test/token",
	})
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("unavailable")),
			Request:    r,
		}, nil
	})}

	_, err := client.Exchange(context.Background(), "code", "verifier", "https://poc.test/callback", "nonce")
	if !errors.Is(err, entity.ErrTelegramProviderUnavailable) {
		t.Fatalf("expected provider unavailable, got %v", err)
	}
}
