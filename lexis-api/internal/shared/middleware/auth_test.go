package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	mw "github.com/lexis-app/lexis-api/internal/shared/middleware"
)

// mockBlacklist implements middleware.Blacklist for testing.
type mockBlacklist struct {
	revoked bool
	err     error
}

func (m *mockBlacklist) IsBlacklisted(_ context.Context, _ string) (bool, error) {
	return m.revoked, m.err
}

func signToken(t *testing.T, secret []byte, claims jwt.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

var testSecret = []byte("test-secret-32-characters-long!!")

func validClaims() jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Subject:   "user-123",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
}

func okHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func unreachableHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	})
}

func TestAuth(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		blacklist  mw.Blacklist
		wantCode   int
	}{
		{
			name:       "valid token without blacklist",
			authHeader: "Bearer " + signToken(t, testSecret, validClaims()),
			blacklist:  nil,
			wantCode:   http.StatusOK,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			blacklist:  nil,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "non-bearer authorization",
			authHeader: "Basic dXNlcjpwYXNz",
			blacklist:  nil,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "expired token",
			authHeader: "Bearer " + signToken(t, testSecret, jwt.RegisteredClaims{Subject: "user-123", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))}),
			blacklist:  nil,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "wrong signing method (RSA token)",
			authHeader: "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTEyMyJ9.invalid",
			blacklist:  nil,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "malformed token string",
			authHeader: "Bearer not-a-jwt",
			blacklist:  nil,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "token without subject",
			authHeader: "Bearer " + signToken(t, testSecret, jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute))}),
			blacklist:  nil,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "blacklisted user (revoked)",
			authHeader: "Bearer " + signToken(t, testSecret, validClaims()),
			blacklist:  &mockBlacklist{revoked: true},
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "blacklist error returns 503",
			authHeader: "Bearer " + signToken(t, testSecret, validClaims()),
			blacklist:  &mockBlacklist{err: errors.New("redis down")},
			wantCode:   http.StatusServiceUnavailable,
		},
		{
			name:       "blacklist returns not revoked",
			authHeader: "Bearer " + signToken(t, testSecret, validClaims()),
			blacklist:  &mockBlacklist{revoked: false},
			wantCode:   http.StatusOK,
		},
		{
			name:       "wrong secret",
			authHeader: "Bearer " + signToken(t, []byte("wrong-secret-32-characters-long!"), validClaims()),
			blacklist:  nil,
			wantCode:   http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var inner http.Handler
			if tc.wantCode == http.StatusOK {
				inner = okHandler(t)
			} else {
				inner = unreachableHandler(t)
			}

			handler := mw.Auth(testSecret, tc.blacklist)(inner)

			req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tc.wantCode, rec.Code)
		})
	}
}

func TestAuth_SetsUserIDInContext(t *testing.T) {
	handler := mw.Auth(testSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := mw.GetUserID(r.Context())
		assert.Equal(t, "user-123", userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, testSecret, validClaims()))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetUserID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "", mw.GetUserID(ctx))
}

func TestGetUserID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), mw.UserIDKey, 12345)
	assert.Equal(t, "", mw.GetUserID(ctx))
}
