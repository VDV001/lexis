package middleware_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mw "github.com/lexis-app/lexis-api/internal/shared/middleware"
)

func TestMaxBodySize_WithinLimit(t *testing.T) {
	handler := mw.MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(body))
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMaxBodySize_ExceedsLimit(t *testing.T) {
	handler := mw.MaxBodySize(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		// Reading more than 5 bytes should produce an error
		assert.Error(t, err)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	}))

	body := strings.Repeat("x", 100)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}
