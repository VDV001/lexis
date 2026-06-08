package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv  string
	AppPort int

	DatabaseURL      string
	DatabaseMaxConns int

	RedisURL      string
	RedisPassword string

	AnthropicAPIKey  string
	QwenAPIKey       string
	OpenAIAPIKey     string
	GeminiAPIKey     string
	OpenRouterAPIKey string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	// LegacyTokenCutoff governs hard rejection of JWTs with no scope claim.
	// Zero value (cutoff disabled) keeps the migration grant active —
	// no-scope tokens receive default user scopes. A non-zero cutoff
	// switches Auth middleware into rejection mode (401) for any token
	// without a scope claim, regardless of iat. See issue #9 + the
	// auth-scopes runbook for the migration timeline.
	LegacyTokenCutoff time.Time

	CORSAllowedOrigins string
	LogLevel           string
}

func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("APP_PORT", "8080"))
	maxConns, _ := strconv.Atoi(getEnv("DATABASE_MAX_CONNS", "25"))

	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}

	var legacyCutoff time.Time
	if raw := os.Getenv("LEGACY_TOKEN_CUTOFF"); raw != "" {
		legacyCutoff, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, fmt.Errorf("invalid LEGACY_TOKEN_CUTOFF (expected RFC3339, e.g. 2026-06-08T00:00:00Z): %w", err)
		}
	}

	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		AppPort:            port,
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://langtutor:langtutor@localhost:5432/langtutor?sslmode=disable"),
		DatabaseMaxConns:   maxConns,
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		AnthropicAPIKey:    getEnv("ANTHROPIC_API_KEY", ""),
		QwenAPIKey:         getEnv("QWEN_API_KEY", ""),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
		GeminiAPIKey:       getEnv("GEMINI_API_KEY", ""),
		OpenRouterAPIKey:   getEnv("OPENROUTER_API_KEY", ""),
		JWTSecret:          getEnv("APP_SECRET", "dev-secret-change-in-production-32ch"),
		JWTAccessTTL:       accessTTL,
		JWTRefreshTTL:      refreshTTL,
		LegacyTokenCutoff:  legacyCutoff,
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}

	if cfg.AppEnv == "production" && cfg.JWTSecret == "dev-secret-change-in-production-32ch" {
		return nil, fmt.Errorf("APP_SECRET must be explicitly set in production")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
