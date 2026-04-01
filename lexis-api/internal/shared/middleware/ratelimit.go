package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

func RateLimit(redisClient *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			key := fmt.Sprintf("ratelimit:%s:%s", r.URL.Path, ip)

			ctx := r.Context()

			script := redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`)
			result, err := script.Run(ctx, redisClient, []string{key}, int(window.Seconds())).Int64()
			if err != nil {
				// If Redis is down, allow request (fail open)
				next.ServeHTTP(w, r)
				return
			}
			count := result

			if count > int64(limit) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func LoginRateLimit(redisClient *redis.Client) func(http.Handler) http.Handler {
	return RateLimit(redisClient, 5, 15*time.Minute)
}
