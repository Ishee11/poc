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
	Tracing     TracingConfig
	Auth        AuthConfig
	Push        PushConfig
	Kafka       KafkaConfig
}

type TracingConfig struct {
	ServiceName  string
	OTLPEndpoint string
	OTLPInsecure bool
}

type AuthConfig struct {
	Enabled                  bool
	CookieName               string
	CookieSecure             bool
	CookieSameSite           string
	SessionTTL               time.Duration
	IdleTTL                  time.Duration
	LoginRateLimit           string
	SeedAdminEmail           string
	SeedAdminPass            string
	SeedUserEmail            string
	SeedUserPass             string
	AppOrigin                string
	TelegramEnabled          bool
	TelegramClientID         string
	TelegramClientSecret     string
	TelegramBotEnabled       bool
	TelegramBotUsername      string
	TelegramBotToken         string
	TelegramBotWebhookSecret string
}

type PushConfig struct {
	Enabled      bool
	Subject      string
	PublicKey    string
	PrivateKey   string
	Warnings     []int64
	PollInterval time.Duration
}

type KafkaConfig struct {
	Enabled         bool
	Brokers         []string
	OutboxTopic     string
	AuditGroupID    string
	OutboxBatchSize int
	OutboxInterval  time.Duration
}

func Load() (*Config, error) {
	appOrigin := os.Getenv("APP_ORIGIN")
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
		Tracing: TracingConfig{
			ServiceName:  getEnv("OTEL_SERVICE_NAME", "poker-app"),
			OTLPEndpoint: strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
			OTLPInsecure: getBoolEnv("OTEL_EXPORTER_OTLP_INSECURE", true),
		},
		Auth: AuthConfig{
			Enabled:                  getBoolEnv("AUTH_ENABLED", true),
			CookieName:               getEnv("AUTH_COOKIE_NAME", "sid"),
			CookieSecure:             getBoolEnv("AUTH_COOKIE_SECURE", true),
			CookieSameSite:           getEnv("AUTH_COOKIE_SAMESITE", "Lax"),
			SessionTTL:               sessionTTL,
			IdleTTL:                  idleTTL,
			LoginRateLimit:           getEnv("AUTH_LOGIN_RATE_LIMIT", "5/min"),
			SeedAdminEmail:           os.Getenv("AUTH_SEED_ADMIN_EMAIL"),
			SeedAdminPass:            os.Getenv("AUTH_SEED_ADMIN_PASSWORD"),
			SeedUserEmail:            os.Getenv("AUTH_SEED_USER_EMAIL"),
			SeedUserPass:             os.Getenv("AUTH_SEED_USER_PASSWORD"),
			AppOrigin:                appOrigin,
			TelegramEnabled:          getBoolEnv("TELEGRAM_OIDC_ENABLED", false),
			TelegramClientID:         strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_CLIENT_ID")),
			TelegramClientSecret:     strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_CLIENT_SECRET")),
			TelegramBotEnabled:       getBoolEnv("TELEGRAM_LOGIN_BOT_ENABLED", false),
			TelegramBotUsername:      strings.TrimPrefix(strings.TrimSpace(os.Getenv("TELEGRAM_LOGIN_BOT_USERNAME")), "@"),
			TelegramBotToken:         strings.TrimSpace(os.Getenv("TELEGRAM_LOGIN_BOT_TOKEN")),
			TelegramBotWebhookSecret: strings.TrimSpace(os.Getenv("TELEGRAM_LOGIN_BOT_WEBHOOK_SECRET")),
		},
		Push: PushConfig{
			Enabled:      getBoolEnv("PUSH_ENABLED", false),
			Subject:      normalizePushSubject(getEnv("PUSH_SUBJECT", "")),
			PublicKey:    getEnv("PUSH_VAPID_PUBLIC_KEY", ""),
			PrivateKey:   getEnv("PUSH_VAPID_PRIVATE_KEY", ""),
			Warnings:     []int64{60, 10},
			PollInterval: getDurationOrDefault("PUSH_POLL_INTERVAL", time.Second),
		},
		Kafka: KafkaConfig{
			Enabled:         getBoolEnv("KAFKA_ENABLED", false),
			Brokers:         getCSVEnv("KAFKA_BROKERS", []string{"127.0.0.1:19092"}),
			OutboxTopic:     getEnv("KAFKA_OUTBOX_TOPIC", "poker.events"),
			AuditGroupID:    getEnv("KAFKA_AUDIT_GROUP_ID", "poker-audit"),
			OutboxBatchSize: getIntEnv("KAFKA_OUTBOX_BATCH_SIZE", 100),
			OutboxInterval:  getDurationOrDefault("KAFKA_OUTBOX_INTERVAL", time.Second),
		},
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if err := validatePushConfig(&cfg.Push); err != nil {
		return nil, err
	}
	if cfg.Auth.TelegramEnabled && (cfg.Auth.TelegramClientID == "" || cfg.Auth.TelegramClientSecret == "" || cfg.Auth.AppOrigin == "") {
		return nil, fmt.Errorf("TELEGRAM_OIDC_CLIENT_ID, TELEGRAM_OIDC_CLIENT_SECRET, and APP_ORIGIN are required when TELEGRAM_OIDC_ENABLED=true")
	}
	if cfg.Auth.TelegramBotEnabled && (cfg.Auth.TelegramBotUsername == "" || cfg.Auth.TelegramBotToken == "" || cfg.Auth.TelegramBotWebhookSecret == "") {
		return nil, fmt.Errorf("TELEGRAM_LOGIN_BOT_USERNAME, TELEGRAM_LOGIN_BOT_TOKEN, and TELEGRAM_LOGIN_BOT_WEBHOOK_SECRET are required when TELEGRAM_LOGIN_BOT_ENABLED=true")
	}

	return cfg, nil
}

func getCSVEnv(key string, def []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	if len(result) == 0 {
		return def
	}

	return result
}

func getIntEnv(key string, def int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}

	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil || parsed <= 0 {
		return def
	}

	return parsed
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
	if strings.HasPrefix(strings.ToLower(value), "https:") {
		return "https:" + value[len("https:"):]
	}
	return value
}
