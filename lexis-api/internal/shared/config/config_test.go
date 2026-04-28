package config_test

import (
	"testing"

	"github.com/lexis-app/lexis-api/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, 8080, cfg.AppPort)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 25, cfg.DatabaseMaxConns)
}

func TestLoad_CustomEnvVars(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DATABASE_MAX_CONNS", "50")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("REDIS_URL", "redis://custom:6380")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("JWT_ACCESS_TTL", "30m")
	t.Setenv("JWT_REFRESH_TTL", "168h")
	t.Setenv("APP_SECRET", "custom-secret-32-chars-long!!!!!")
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-xxx")
	t.Setenv("QWEN_API_KEY", "qwen-xxx")
	t.Setenv("OPENAI_API_KEY", "sk-xxx")
	t.Setenv("GEMINI_API_KEY", "gem-xxx")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "staging", cfg.AppEnv)
	assert.Equal(t, 9090, cfg.AppPort)
	assert.Equal(t, 50, cfg.DatabaseMaxConns)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "redis://custom:6380", cfg.RedisURL)
	assert.Equal(t, "secret", cfg.RedisPassword)
	assert.Equal(t, "https://app.example.com", cfg.CORSAllowedOrigins)
	assert.Equal(t, "custom-secret-32-chars-long!!!!!", cfg.JWTSecret)
	assert.Equal(t, "sk-ant-xxx", cfg.AnthropicAPIKey)
	assert.Equal(t, "qwen-xxx", cfg.QwenAPIKey)
	assert.Equal(t, "sk-xxx", cfg.OpenAIAPIKey)
	assert.Equal(t, "gem-xxx", cfg.GeminiAPIKey)
}

func TestLoad_ProductionRequiresSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	// Do NOT set APP_SECRET => uses default => should fail

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_SECRET must be explicitly set in production")
}

func TestLoad_ProductionWithCustomSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_SECRET", "my-prod-secret-at-least-32-chars!")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "my-prod-secret-at-least-32-chars!", cfg.JWTSecret)
}

func TestLoad_InvalidAccessTTL(t *testing.T) {
	t.Setenv("JWT_ACCESS_TTL", "not-a-duration")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JWT_ACCESS_TTL")
}

func TestLoad_InvalidRefreshTTL(t *testing.T) {
	t.Setenv("JWT_REFRESH_TTL", "bad")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JWT_REFRESH_TTL")
}

func TestLoad_InvalidPortFallsBackToZero(t *testing.T) {
	t.Setenv("APP_PORT", "not-a-number")

	cfg, err := config.Load()
	require.NoError(t, err)
	// strconv.Atoi returns 0 on error, and the code ignores the error
	assert.Equal(t, 0, cfg.AppPort)
}

func TestLoad_InvalidMaxConnsFallsBackToZero(t *testing.T) {
	t.Setenv("DATABASE_MAX_CONNS", "abc")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.DatabaseMaxConns)
}

func TestLoad_DefaultDurations(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 15*60*1e9, float64(cfg.JWTAccessTTL))  // 15m in nanoseconds
	assert.Equal(t, 720*3600*1e9, float64(cfg.JWTRefreshTTL)) // 720h in nanoseconds
}

func TestLoad_EmptyEnvVarUsesDefault(t *testing.T) {
	// Setting env var to empty string should cause getEnv to return the default
	// because the code checks `val != ""`.
	t.Setenv("APP_ENV", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "development", cfg.AppEnv)
}
