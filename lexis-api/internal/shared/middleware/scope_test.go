package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

// signWithScopes builds an HS256 token carrying sub + exp + the given
// scope strings, mirroring what AuthService.generateAccessToken now emits.
func signWithScopes(t *testing.T, sub string, scopes []string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   sub,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"scope": scopes,
		"aud":   "lexis-api",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(testSecret)
	require.NoError(t, err)
	return signed
}

func TestAuth_putsScopesIntoRequestContext(t *testing.T) {
	want := []domain.Scope{
		domain.ScopeChatRead,
		domain.ScopeVocabWrite,
	}
	scopeStrings := []string{string(want[0]), string(want[1])}
	raw := signWithScopes(t, "user-1", scopeStrings)

	var got []domain.Scope
	handler := middleware.Auth(testSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = middleware.GetScopes(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.ElementsMatch(t, want, got)
}

func TestAuth_emptyScopesContextWhenClaimAbsent(t *testing.T) {
	// Legacy-style token: no scope claim. Middleware must NOT crash —
	// the post-cutoff legacy-fallback behaviour lands in a separate cycle.
	// Here we only assert the contract that GetScopes returns an empty
	// slice when nothing is in the JWT.
	claims := jwt.MapClaims{
		"sub": "legacy-user",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := tok.SignedString(testSecret)
	require.NoError(t, err)

	var got []domain.Scope
	var sawHandler bool
	handler := middleware.Auth(testSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = middleware.GetScopes(r.Context())
		sawHandler = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.True(t, sawHandler, "middleware must not block legacy tokens at this stage")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, got)
}
