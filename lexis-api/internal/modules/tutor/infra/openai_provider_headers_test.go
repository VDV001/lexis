package infra

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureHeadersServer(t *testing.T, captured *http.Header) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{}"}}]}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestOpenAICompatibleProvider_AppliesExtraHeaders(t *testing.T) {
	var got http.Header
	srv := captureHeadersServer(t, &got)

	p := NewOpenAICompatibleProviderWithHeaders("test-key", srv.URL, map[string]string{
		"HTTP-Referer": "https://github.com/VDV001/lexis",
		"X-Title":      "Lexis",
	})

	_, err := p.GenerateExercise(context.Background(), domain.ExerciseRequest{
		Model:  "openai/gpt-4o-mini",
		System: "system",
	})
	require.NoError(t, err)

	assert.Equal(t, "https://github.com/VDV001/lexis", got.Get("HTTP-Referer"))
	assert.Equal(t, "Lexis", got.Get("X-Title"))
	assert.Equal(t, "Bearer test-key", got.Get("Authorization"))
}

func TestOpenAICompatibleProvider_NoExtraHeadersByDefault(t *testing.T) {
	var got http.Header
	srv := captureHeadersServer(t, &got)

	p := NewOpenAICompatibleProvider("test-key", srv.URL)

	_, err := p.GenerateExercise(context.Background(), domain.ExerciseRequest{
		Model:  "gpt-4o",
		System: "system",
	})
	require.NoError(t, err)

	// OpenAI / Qwen instances must not leak OpenRouter-specific headers.
	assert.Empty(t, got.Get("HTTP-Referer"))
	assert.Empty(t, got.Get("X-Title"))
	assert.Equal(t, "Bearer test-key", got.Get("Authorization"))
}
