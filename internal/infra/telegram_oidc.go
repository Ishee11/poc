package infra

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type TelegramOIDCClientConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	JWKSURL      string
}

type TelegramOIDCClient struct {
	config TelegramOIDCClientConfig
	client *http.Client
	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey
	keysAt time.Time
}

func NewTelegramOIDCClient(config TelegramOIDCClientConfig) *TelegramOIDCClient {
	if config.TokenURL == "" {
		config.TokenURL = "https://oauth.telegram.org/token"
	}
	if config.JWKSURL == "" {
		config.JWKSURL = "https://oauth.telegram.org/.well-known/jwks.json"
	}
	return &TelegramOIDCClient{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
		keys:   make(map[string]*rsa.PublicKey),
	}
}

func (c *TelegramOIDCClient) Exchange(
	ctx context.Context,
	code string,
	codeVerifier string,
	redirectURI string,
	nonce string,
) (usecase.TelegramOIDCClaims, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {c.config.ClientID},
		"code_verifier": {codeVerifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return usecase.TelegramOIDCClaims{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.config.ClientID, c.config.ClientSecret)
	resp, err := c.client.Do(req)
	if err != nil {
		return usecase.TelegramOIDCClaims{}, fmt.Errorf("%w: token request: %v", entity.ErrTelegramProviderUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			return usecase.TelegramOIDCClaims{}, fmt.Errorf("%w: token endpoint status %d", entity.ErrTelegramProviderUnavailable, resp.StatusCode)
		}
		return usecase.TelegramOIDCClaims{}, fmt.Errorf("telegram token endpoint status %d", resp.StatusCode)
	}
	var tokenResponse struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tokenResponse); err != nil {
		return usecase.TelegramOIDCClaims{}, err
	}
	return c.validateIDToken(ctx, tokenResponse.IDToken, nonce)
}

type telegramJWTHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

type telegramJWTClaims struct {
	Issuer            string          `json:"iss"`
	Audience          json.RawMessage `json:"aud"`
	Subject           string          `json:"sub"`
	TelegramUserID    int64           `json:"id"`
	ExpiresAt         int64           `json:"exp"`
	NotBefore         int64           `json:"nbf"`
	Nonce             string          `json:"nonce"`
	Name              string          `json:"name"`
	PreferredUsername string          `json:"preferred_username"`
	Picture           string          `json:"picture"`
}

func (c *TelegramOIDCClient) validateIDToken(ctx context.Context, rawToken string, nonce string) (usecase.TelegramOIDCClaims, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return usecase.TelegramOIDCClaims{}, errors.New("invalid jwt structure")
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return usecase.TelegramOIDCClaims{}, err
	}
	claimsRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return usecase.TelegramOIDCClaims{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return usecase.TelegramOIDCClaims{}, err
	}
	var header telegramJWTHeader
	if err := json.Unmarshal(headerRaw, &header); err != nil || header.Algorithm != "RS256" || header.KeyID == "" {
		return usecase.TelegramOIDCClaims{}, errors.New("unsupported jwt header")
	}
	key, err := c.publicKey(ctx, header.KeyID, false)
	if err != nil {
		key, err = c.publicKey(ctx, header.KeyID, true)
	}
	if err != nil {
		return usecase.TelegramOIDCClaims{}, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return usecase.TelegramOIDCClaims{}, errors.New("invalid jwt signature")
	}
	var claims telegramJWTClaims
	if err := json.Unmarshal(claimsRaw, &claims); err != nil {
		return usecase.TelegramOIDCClaims{}, err
	}
	now := time.Now().Unix()
	if claims.Issuer != "https://oauth.telegram.org" || claims.Subject == "" || claims.TelegramUserID <= 0 ||
		claims.ExpiresAt <= now || (claims.NotBefore != 0 && claims.NotBefore > now+30) ||
		claims.Nonce != nonce || !audienceContains(claims.Audience, c.config.ClientID) {
		return usecase.TelegramOIDCClaims{}, errors.New("invalid jwt claims")
	}
	return usecase.TelegramOIDCClaims{
		Subject: fmt.Sprint(claims.TelegramUserID), LegacySubject: claims.Subject, Username: claims.PreferredUsername,
		DisplayName: claims.Name, PictureURL: claims.Picture,
	}, nil
}

func audienceContains(raw json.RawMessage, expected string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single == expected
	}
	var multiple []string
	if json.Unmarshal(raw, &multiple) != nil {
		return false
	}
	for _, audience := range multiple {
		if audience == expected {
			return true
		}
	}
	return false
}

func (c *TelegramOIDCClient) publicKey(ctx context.Context, keyID string, force bool) (*rsa.PublicKey, error) {
	c.mu.RLock()
	key := c.keys[keyID]
	fresh := time.Since(c.keysAt) < time.Hour
	c.mu.RUnlock()
	if key != nil && fresh && !force {
		return key, nil
	}
	if err := c.refreshKeys(ctx); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	key = c.keys[keyID]
	if key == nil {
		return nil, errors.New("telegram signing key not found")
	}
	return key, nil
}

func (c *TelegramOIDCClient) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.JWKSURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: jwks request: %v", entity.ErrTelegramProviderUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("%w: jwks endpoint status %d", entity.ErrTelegramProviderUnavailable, resp.StatusCode)
		}
		return fmt.Errorf("telegram jwks endpoint status %d", resp.StatusCode)
	}
	var jwks struct {
		Keys []struct {
			KeyID string `json:"kid"`
			Type  string `json:"kty"`
			Use   string `json:"use"`
			N     string `json:"n"`
			E     string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&jwks); err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range jwks.Keys {
		if item.KeyID == "" || item.Type != "RSA" || item.N == "" || item.E == "" {
			continue
		}
		nRaw, nErr := base64.RawURLEncoding.DecodeString(item.N)
		eRaw, eErr := base64.RawURLEncoding.DecodeString(item.E)
		if nErr != nil || eErr != nil || len(eRaw) == 0 || len(eRaw) > 4 {
			continue
		}
		e := 0
		for _, b := range eRaw {
			e = e<<8 + int(b)
		}
		if e <= 0 {
			continue
		}
		keys[item.KeyID] = &rsa.PublicKey{N: new(big.Int).SetBytes(nRaw), E: e}
	}
	if len(keys) == 0 {
		return errors.New("telegram jwks contains no rsa keys")
	}
	c.mu.Lock()
	c.keys = keys
	c.keysAt = time.Now()
	c.mu.Unlock()
	return nil
}
