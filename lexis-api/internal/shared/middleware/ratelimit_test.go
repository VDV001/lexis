package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	mw "github.com/lexis-app/lexis-api/internal/shared/middleware"
)

func testRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	addrs := []string{"localhost:6379", "192.168.97.3:6379"}
	if addr := os.Getenv("TEST_REDIS_ADDR"); addr != "" {
		addrs = []string{addr}
	}

	for _, addr := range addrs {
		client := redis.NewClient(&redis.Options{
			Addr:        addr,
			DialTimeout: 500 * time.Millisecond,
		})
		if err := client.Ping(context.Background()).Err(); err == nil {
			t.Cleanup(func() { client.Close() })
			return client
		}
		client.Close()
	}
	t.Skip("Redis not available (set TEST_REDIS_ADDR to override)")
	return nil
}

func deadRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:        "localhost:1",
		DialTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() { client.Close() })
	return client
}

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	client := testRedisClient(t)
	client.FlushDB(context.Background())

	handler := mw.RateLimit(client, "test_allow", 5, time.Minute)(okHandler(t))

	for i := range 5 {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should pass", i+1)
	}
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	client := testRedisClient(t)
	client.FlushDB(context.Background())

	handler := mw.RateLimit(client, "test_block", 2, time.Minute)(okHandler(t))

	for range 2 {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestRateLimit_DifferentIPsAreIndependent(t *testing.T) {
	client := testRedisClient(t)
	client.FlushDB(context.Background())

	handler := mw.RateLimit(client, "test_ips", 1, time.Minute)(okHandler(t))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:1111"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.2:2222"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_FailsOpenWhenRedisDown(t *testing.T) {
	client := deadRedisClient(t)

	handler := mw.RateLimit(client, "test", 1, time.Minute)(okHandler(t))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "should fail open when Redis is unavailable")
}

func TestRateLimit_RemoteAddrWithoutPort(t *testing.T) {
	client := testRedisClient(t)
	client.FlushDB(context.Background())

	handler := mw.RateLimit(client, "test_noport", 1, time.Minute)(okHandler(t))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_EmptyIPFallback(t *testing.T) {
	client := testRedisClient(t)
	client.FlushDB(context.Background())

	handler := mw.RateLimit(client, "test_emptyip", 1, time.Minute)(okHandler(t))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.RemoteAddr = ":0"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.RemoteAddr = ":0"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestLoginRateLimit_FailsClosedWhenRedisDown(t *testing.T) {
	client := deadRedisClient(t)

	handler := mw.LoginRateLimit(client)(okHandler(t))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "login should fail closed when Redis is unavailable")
}

func TestLoginRateLimit_AllowsWithinLimit(t *testing.T) {
	client := testRedisClient(t)
	client.FlushDB(context.Background())

	handler := mw.LoginRateLimit(client)(okHandler(t))

	for i := range 5 {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", nil)
		req.RemoteAddr = "10.0.0.50:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should pass", i+1)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", nil)
	req.RemoteAddr = "10.0.0.50:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
