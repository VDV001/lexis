package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	mw "github.com/lexis-app/lexis-api/internal/shared/middleware"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name            string
		allowedOrigins  string
		requestOrigin   string
		method          string
		wantCORSHeaders bool
		wantCode        int
	}{
		{
			name:            "allowed origin GET",
			allowedOrigins:  "http://localhost:3000",
			requestOrigin:   "http://localhost:3000",
			method:          http.MethodGet,
			wantCORSHeaders: true,
			wantCode:        http.StatusOK,
		},
		{
			name:            "disallowed origin GET",
			allowedOrigins:  "http://localhost:3000",
			requestOrigin:   "http://evil.com",
			method:          http.MethodGet,
			wantCORSHeaders: false,
			wantCode:        http.StatusOK,
		},
		{
			name:            "preflight OPTIONS allowed origin",
			allowedOrigins:  "http://localhost:3000",
			requestOrigin:   "http://localhost:3000",
			method:          http.MethodOptions,
			wantCORSHeaders: true,
			wantCode:        http.StatusNoContent,
		},
		{
			name:            "preflight OPTIONS disallowed origin",
			allowedOrigins:  "http://localhost:3000",
			requestOrigin:   "http://evil.com",
			method:          http.MethodOptions,
			wantCORSHeaders: false,
			wantCode:        http.StatusNoContent,
		},
		{
			name:            "multiple allowed origins, second matches",
			allowedOrigins:  "http://localhost:3000, http://app.example.com",
			requestOrigin:   "http://app.example.com",
			method:          http.MethodGet,
			wantCORSHeaders: true,
			wantCode:        http.StatusOK,
		},
		{
			name:            "no origin header",
			allowedOrigins:  "http://localhost:3000",
			requestOrigin:   "",
			method:          http.MethodGet,
			wantCORSHeaders: false,
			wantCode:        http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := mw.CORS(tc.allowedOrigins)(inner)

			req := httptest.NewRequestWithContext(context.Background(), tc.method, "/test", nil)
			if tc.requestOrigin != "" {
				req.Header.Set("Origin", tc.requestOrigin)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantCode, rec.Code)

			if tc.wantCORSHeaders {
				assert.Equal(t, tc.requestOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
				assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
				assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
				assert.Equal(t, "Origin", rec.Header().Get("Vary"))
			} else {
				assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}
