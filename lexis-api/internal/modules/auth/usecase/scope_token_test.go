package usecase_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

// parseAccessToken decodes an access token signed by newTestService back
// into MapClaims and verifies the signature against testJWTSecret.
func parseAccessToken(t *testing.T, raw string) jwt.MapClaims {
	t.Helper()
	parsed, err := jwt.Parse(raw, func(*jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid, "access token must validate against the test secret")
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	return claims
}

// audClaim normalises the "aud" field which jwt-go represents as either
// a string (single audience) or an []interface{} (multiple). For Lexis
// we always issue exactly one audience, but parsing must accept both.
func audClaim(t *testing.T, claims jwt.MapClaims) []string {
	t.Helper()
	switch v := claims["aud"].(type) {
	case nil:
		return nil
	case string:
		return []string{v}
	case []interface{}:
		out := make([]string, len(v))
		for i, x := range v {
			s, ok := x.(string)
			require.True(t, ok)
			out[i] = s
		}
		return out
	default:
		t.Fatalf("unexpected aud claim type: %T", v)
		return nil
	}
}

func TestLogin_AccessTokenIncludesDefaultScopesAndAudience(t *testing.T) {
	svc, users, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	users.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: string(hash),
		DisplayName:  "Test User",
	}, nil)
	tokens.EXPECT().CreateRefreshToken(ctx, gomock.Any()).Return(nil)

	result, err := svc.Login(ctx, "test@example.com", "password123", "ua", "ip")
	require.NoError(t, err)

	claims := parseAccessToken(t, result.AccessToken)

	rawScopes, ok := claims["scope"].([]interface{})
	require.True(t, ok, "access token must carry a scope claim that is an array")
	require.NotEmpty(t, rawScopes, "scope claim must not be empty for a normal login")

	got := make(map[string]bool, len(rawScopes))
	for _, s := range rawScopes {
		str, ok := s.(string)
		require.True(t, ok, "scope claim entry must be a string, got %T", s)
		got[str] = true
	}

	for _, expected := range domain.DefaultUserScopes() {
		assert.Truef(t, got[string(expected)],
			"login token missing default scope %q (got %v)", expected, got)
	}
	assert.Falsef(t, got[string(domain.ScopeAdminFull)],
		"login token leaked %q — admin must require explicit grant", domain.ScopeAdminFull)

	aud := audClaim(t, claims)
	assert.Equal(t, []string{"lexis-api"}, aud,
		`access token aud must be ["lexis-api"]`)
}

func TestRegister_AccessTokenIncludesDefaultScopesAndAudience(t *testing.T) {
	svc, users, tokens, settings, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
		u.ID = "user-456"
		return nil
	})
	settings.EXPECT().Upsert(ctx, gomock.Any()).Return(nil)
	tokens.EXPECT().CreateRefreshToken(ctx, gomock.Any()).Return(nil)

	result, err := svc.Register(ctx, "fresh@example.com", "password123", "Fresh")
	require.NoError(t, err)

	claims := parseAccessToken(t, result.AccessToken)

	rawScopes, ok := claims["scope"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, rawScopes)

	got := make(map[string]bool, len(rawScopes))
	for _, s := range rawScopes {
		str, ok := s.(string)
		require.True(t, ok, "scope claim entry must be a string, got %T", s)
		got[str] = true
	}
	for _, expected := range domain.DefaultUserScopes() {
		assert.Truef(t, got[string(expected)], "register token missing %q", expected)
	}
	assert.False(t, got[string(domain.ScopeAdminFull)])

	assert.Equal(t, []string{"lexis-api"}, audClaim(t, claims))
}
