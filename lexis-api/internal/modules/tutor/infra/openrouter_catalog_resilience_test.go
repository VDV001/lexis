package infra

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCatalog_StaleWhileError: once warmed, a later fetch failure must serve the
// last good cache instead of erroring.
func TestCatalog_StaleWhileError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, sampleOpenRouterModels)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// Tiny TTL so the second call is past expiry and triggers a refetch.
	src := newOpenRouterCatalogSource("k", srv.URL, time.Millisecond)

	first, err := src.List(context.Background())
	require.NoError(t, err)
	require.Len(t, first, 2)

	time.Sleep(5 * time.Millisecond)

	second, err := src.List(context.Background())
	require.NoError(t, err, "stale cache must be served when refresh fails")
	assert.Equal(t, first, second, "served data must be the last good cache")
	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2), "a refetch must have been attempted")
}

// TestCatalog_NoCacheStillErrors: with no prior cache, a fetch failure must
// still surface an error (nothing stale to serve).
func TestCatalog_NoCacheStillErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src := newOpenRouterCatalogSource("k", srv.URL, time.Hour)
	_, err := src.List(context.Background())
	require.Error(t, err)
}

// TestCatalog_Singleflight: concurrent callers on a cold cache collapse into a
// single upstream fetch.
func TestCatalog_Singleflight(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(20 * time.Millisecond) // widen the race window
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, sampleOpenRouterModels)
	}))
	defer srv.Close()

	src := newOpenRouterCatalogSource("k", srv.URL, time.Hour)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = src.List(context.Background())
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls),
		"concurrent cold-cache reads must collapse into one upstream fetch")
}
