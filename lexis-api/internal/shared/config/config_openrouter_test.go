package config_test

import (
	"testing"

	"github.com/lexis-app/lexis-api/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_OpenRouterAPIKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "sk-or-v1-test")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "sk-or-v1-test", cfg.OpenRouterAPIKey)
}

func TestLoad_OpenRouterAPIKey_DefaultsEmpty(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.OpenRouterAPIKey,
		"OpenRouter key must default to empty so the provider stays unregistered")
}
