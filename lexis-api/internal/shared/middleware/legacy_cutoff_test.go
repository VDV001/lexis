package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

// signLegacy builds a token with no scope claim, optionally pinning iat
// to the supplied time. Mirrors what AuthService used to emit before the
// scope-aware token generator landed.
func signLegacy(t *testing.T, sub string, iat time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	if !iat.IsZero() {
		claims["iat"] = iat.Unix()
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(testSecret)
	require.NoError(t, err)
	return signed
}

func TestAuth_legacyCutoff(t *testing.T) {
	cutoff := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name       string
		cutoff     time.Time
		token      func(t *testing.T) string
		wantStatus int
	}{
		{
			name:   "cutoff active + legacy token (iat before cutoff) -> 401",
			cutoff: cutoff,
			token: func(t *testing.T) string {
				return signLegacy(t, "u-old", cutoff.Add(-30*24*time.Hour))
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "cutoff active + legacy token (iat after cutoff) -> 401 (issuer regression)",
			cutoff: cutoff,
			token: func(t *testing.T) string {
				return signLegacy(t, "u-bug", cutoff.Add(time.Hour))
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "cutoff active + legacy token (no iat at all) -> 401",
			cutoff: cutoff,
			token: func(t *testing.T) string {
				return signLegacy(t, "u-noiat", time.Time{})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "cutoff active + scoped token -> 200 (cutoff only affects no-scope tokens)",
			cutoff: cutoff,
			token: func(t *testing.T) string {
				return signWithScopes(t, "u-scoped", []string{"chat:read"})
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "cutoff disabled (zero) + legacy token -> 200 (regression: migration grant still works)",
			cutoff: time.Time{},
			token: func(t *testing.T) string {
				return signLegacy(t, "u-mig", time.Now().Add(-24*time.Hour))
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := middleware.Auth(testSecret, nil, tc.cutoff)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token(t))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code, rec.Body.String())
		})
	}
}
