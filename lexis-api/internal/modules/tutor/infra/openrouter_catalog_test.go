package infra

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleOpenRouterModels = `{
  "data": [
    {
      "id": "openai/gpt-4o-mini",
      "name": "OpenAI: GPT-4o-mini",
      "description": "Cheap and fast",
      "architecture": {"input_modalities": ["text", "image"], "output_modalities": ["text"], "modality": "text+image->text"}
    },
    {
      "id": "openai/dall-e-3",
      "name": "OpenAI: DALL-E 3",
      "description": "Image gen",
      "architecture": {"input_modalities": ["text"], "output_modalities": ["image"], "modality": "text->image"}
    }
  ]
}`

func TestOpenRouterCatalogSource_FetchAndMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, sampleOpenRouterModels)
	}))
	defer srv.Close()

	src := newOpenRouterCatalogSource("test-key", srv.URL, time.Hour)

	models, err := src.List(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 2)

	assert.Equal(t, "openai/gpt-4o-mini", models[0].ID)
	assert.Equal(t, "OpenAI: GPT-4o-mini", models[0].Name)
	assert.Equal(t, "Cheap and fast", models[0].Description)
	assert.Equal(t, []string{"text", "image"}, models[0].InputModalities)
	assert.Equal(t, []string{"text"}, models[0].OutputModalities)
}

func TestOpenRouterCatalogSource_CachesWithinTTL(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, sampleOpenRouterModels)
	}))
	defer srv.Close()

	src := newOpenRouterCatalogSource("test-key", srv.URL, time.Hour)

	_, err := src.List(context.Background())
	require.NoError(t, err)
	_, err = src.List(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls),
		"second List within TTL must be served from cache, not refetched")
}

func TestOpenRouterCatalogSource_FetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":"boom"}`)
	}))
	defer srv.Close()

	src := newOpenRouterCatalogSource("test-key", srv.URL, time.Hour)

	_, err := src.List(context.Background())
	require.Error(t, err)
}
