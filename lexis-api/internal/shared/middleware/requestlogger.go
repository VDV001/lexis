package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger logs one structured record per HTTP request using the given
// slog logger. It replaces chi's text Logger so request logs share the app's
// structured format (JSON in production).
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			defer func() {
				logger.Info("http_request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
					"request_id", chiMiddleware.GetReqID(r.Context()),
					"remote", r.RemoteAddr,
				)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}
