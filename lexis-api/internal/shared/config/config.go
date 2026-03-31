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

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool

	AnthropicAPIKey string
	QwenAPIKey      string
	OpenAIAPIKey    string
	GeminiAPIKey    string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	CORSAllowedOrigins string
	LogLevel            string
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

	return &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		AppPort:            port,
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://langtutor:langtutor@localhost:5432/langtutor?sslmode=disable"),
		DatabaseMaxConns:   maxConns,
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		MinioEndpoint:      getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:     getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseSSL:        getEnv("MINIO_USE_SSL", "false") == "true",
		AnthropicAPIKey:    getEnv("ANTHROPIC_API_KEY", ""),
		QwenAPIKey:         getEnv("QWEN_API_KEY", ""),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
		GeminiAPIKey:       getEnv("GEMINI_API_KEY", ""),
		JWTSecret:          getEnv("APP_SECRET", "dev-secret-change-in-production-32ch"),
		JWTAccessTTL:       accessTTL,
		JWTRefreshTTL:      refreshTTL,
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
