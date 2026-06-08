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

func TestIsTransientStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
		{http.StatusOK, false},
		{http.StatusUnauthorized, false},
		{http.StatusNotFound, false},
		{http.StatusBadRequest, false},
	}
	for _, tt := range tests {
		if got := isTransientStatus(tt.code); got != tt.want {
			t.Errorf("isTransientStatus(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

// TestCatalog_RetriesTransient: a transient 503 is retried with backoff and the
// fetch eventually succeeds.
func TestCatalog_RetriesTransient(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, sampleOpenRouterModels)
	}))
	defer srv.Close()

	src := newOpenRouterCatalogSource("k", srv.URL, time.Hour)
	src.retryBackoff = 0 // no sleep in tests
	src.maxRetries = 3

	models, err := src.List(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls), "two 503s should be retried, third call succeeds")
}

// TestCatalog_DoesNotRetryNonTransient: a 401 is not retried.
func TestCatalog_DoesNotRetryNonTransient(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	src := newOpenRouterCatalogSource("k", srv.URL, time.Hour)
	src.retryBackoff = 0
	src.maxRetries = 3

	_, err := src.List(context.Background())
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "non-transient status must not be retried")
}
