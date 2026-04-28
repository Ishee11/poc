package app

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	LogLevel    string
	Auth        AuthConfig
	Push        PushConfig
}

type AuthConfig struct {
	Enabled        bool
	CookieName     string
	CookieSecure   bool
	CookieSameSite string
	SessionTTL     time.Duration
	IdleTTL        time.Duration
	LoginRateLimit string
	SeedAdminEmail string
	SeedAdminPass  string
	SeedUserEmail  string
	SeedUserPass   string
	AppOrigin      string
}

type PushConfig struct {
	Enabled      bool
	Subject      string
	PublicKey    string
	PrivateKey   string
	Warnings     []int64
	PollInterval time.Duration
}

func Load() (*Config, error) {
	sessionTTL, err := getDurationEnv("AUTH_SESSION_TTL", 12*time.Hour)
	if err != nil {
		return nil, err
	}

	idleTTL, err := getDurationEnv("AUTH_IDLE_TTL", 2*time.Hour)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Auth: AuthConfig{
			Enabled:        getBoolEnv("AUTH_ENABLED", false),
			CookieName:     getEnv("AUTH_COOKIE_NAME", "sid"),
			CookieSecure:   getBoolEnv("AUTH_COOKIE_SECURE", true),
			CookieSameSite: getEnv("AUTH_COOKIE_SAMESITE", "Lax"),
			SessionTTL:     sessionTTL,
			IdleTTL:        idleTTL,
			LoginRateLimit: getEnv("AUTH_LOGIN_RATE_LIMIT", "5/min"),
			SeedAdminEmail: os.Getenv("AUTH_SEED_ADMIN_EMAIL"),
			SeedAdminPass:  os.Getenv("AUTH_SEED_ADMIN_PASSWORD"),
			SeedUserEmail:  os.Getenv("AUTH_SEED_USER_EMAIL"),
			SeedUserPass:   os.Getenv("AUTH_SEED_USER_PASSWORD"),
			AppOrigin:      os.Getenv("APP_ORIGIN"),
		},
		Push: PushConfig{
			Enabled:      getBoolEnv("PUSH_ENABLED", false),
			Subject:      normalizePushSubject(getEnv("PUSH_SUBJECT", "")),
			PublicKey:    getEnv("PUSH_VAPID_PUBLIC_KEY", ""),
			PrivateKey:   getEnv("PUSH_VAPID_PRIVATE_KEY", ""),
			Warnings:     []int64{60, 10},
			PollInterval: getDurationOrDefault("PUSH_POLL_INTERVAL", time.Second),
		},
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getDurationOrDefault(key string, def time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return def
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return def
	}

	return duration
}

func getBoolEnv(key string, def bool) bool {
	switch os.Getenv(key) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return def
	}
}

func getDurationEnv(key string, def time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return def, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s is invalid: %w", key, err)
	}

	return duration, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func normalizePushSubject(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "mailto:") {
		return strings.TrimSpace(value[len("mailto:"):])
	}
	return value
}
