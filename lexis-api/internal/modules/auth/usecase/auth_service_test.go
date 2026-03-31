package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/auth/usecase"
)

func newTestService(t *testing.T) (
	*usecase.AuthService,
	*MockUserRepository,
	*MockTokenRepository,
	*MockSettingsRepository,
	*MockBlacklist,
) {
	ctrl := gomock.NewController(t)
	users := NewMockUserRepository(ctrl)
	tokens := NewMockTokenRepository(ctrl)
	settings := NewMockSettingsRepository(ctrl)
	blacklist := NewMockBlacklist(ctrl)

	svc := usecase.NewAuthService(
		users, tokens, settings, blacklist,
		"test-secret-32-characters-long!!", 15*time.Minute, 720*time.Hour,
	)
	return svc, users, tokens, settings, blacklist
}

func TestRegister_Success(t *testing.T) {
	svc, users, tokens, settings, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
		u.ID = "user-123"
		return nil
	})
	settings.EXPECT().Upsert(ctx, gomock.Any()).Return(nil)
	tokens.EXPECT().CreateRefreshToken(ctx, gomock.Any()).Return(nil)

	result, err := svc.Register(ctx, "test@example.com", "password123", "Test User")
	require.NoError(t, err)
	assert.Equal(t, "user-123", result.User.ID)
	assert.Equal(t, "test@example.com", result.User.Email)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	// Verify password is hashed with bcrypt
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(result.User.PasswordHash), []byte("password123")))
}

func TestRegister_EmailTaken(t *testing.T) {
	svc, users, _, _, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().Create(ctx, gomock.Any()).Return(domain.ErrEmailTaken)

	result, err := svc.Register(ctx, "taken@example.com", "password123", "Test")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestLogin_Success(t *testing.T) {
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

	result, err := svc.Login(ctx, "test@example.com", "password123", "Mozilla/5.0", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "user-123", result.User.ID)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, users, _, _, _ := newTestService(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), 12)
	users.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
		ID:           "user-123",
		PasswordHash: string(hash),
	}, nil)

	result, err := svc.Login(ctx, "test@example.com", "wrongpass", "", "")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, users, _, _, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().GetByEmail(ctx, "nobody@example.com").Return(nil, domain.ErrUserNotFound)

	result, err := svc.Login(ctx, "nobody@example.com", "pass", "", "")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestRefresh_Success(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		UserID:    "user-123",
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	tokens.EXPECT().RevokeByHash(ctx, gomock.Any()).Return(nil)
	tokens.EXPECT().CreateRefreshToken(ctx, gomock.Any()).Return(nil)

	result, err := svc.Refresh(ctx, "raw-refresh-token")
	require.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
}

func TestRefresh_Expired(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		UserID:    "user-123",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}, nil)

	result, err := svc.Refresh(ctx, "expired-token")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrTokenExpired)
}

func TestRefresh_Revoked(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	now := time.Now()
	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		UserID:    "user-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		RevokedAt: &now,
	}, nil)

	result, err := svc.Refresh(ctx, "revoked-token")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrTokenRevoked)
}

func TestLogout_Success(t *testing.T) {
	svc, _, tokens, _, blacklist := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	tokens.EXPECT().RevokeByHash(ctx, gomock.Any()).Return(nil)
	blacklist.EXPECT().Add(ctx, gomock.Any(), gomock.Any()).Return(nil)

	err := svc.Logout(ctx, "raw-refresh-token")
	assert.NoError(t, err)
}

func TestLogoutAll_Success(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().RevokeAllForUser(ctx, "user-123").Return(nil)

	err := svc.LogoutAll(ctx, "user-123")
	assert.NoError(t, err)
}
