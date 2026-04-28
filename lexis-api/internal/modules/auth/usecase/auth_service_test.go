package usecase_test

import (
	"context"
	"errors"
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

func TestRefresh_Revoked_RevokesAllTokens(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	now := time.Now()
	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		UserID:    "user-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		RevokedAt: &now,
	}, nil)
	tokens.EXPECT().RevokeAllForUser(ctx, "user-123").Return(nil)

	result, err := svc.Refresh(ctx, "revoked-token")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrTokenRevoked)
}

func TestRefresh_ConcurrentReuse_RevokesAllTokens(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	// Token appears valid in GetByHash, but RevokeByHash fails because
	// a concurrent request already revoked it.
	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		UserID:    "user-123",
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	tokens.EXPECT().RevokeByHash(ctx, gomock.Any()).Return(domain.ErrTokenNotFound)
	tokens.EXPECT().RevokeAllForUser(ctx, "user-123").Return(nil)

	result, err := svc.Refresh(ctx, "reused-token")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrTokenRevoked)
}

func TestLogout_Success(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().RevokeByHash(ctx, gomock.Any()).Return(nil)

	err := svc.Logout(ctx, "raw-refresh-token")
	assert.NoError(t, err)
}

func TestLogoutAll_Success(t *testing.T) {
	svc, _, tokens, _, blacklist := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().RevokeAllForUser(ctx, "user-123").Return(nil)
	blacklist.EXPECT().Add(ctx, "user_revoked:user-123", 15*time.Minute).Return(nil)

	err := svc.LogoutAll(ctx, "user-123")
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Register — uncovered error paths
// ---------------------------------------------------------------------------

func TestRegister_InvalidPassword(t *testing.T) {
	svc, _, _, _, _ := newTestService(t)
	ctx := context.Background()

	result, err := svc.Register(ctx, "test@example.com", "short", "Test User")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrInvalidPassword)
}

func TestRegister_InvalidEmail(t *testing.T) {
	svc, _, _, _, _ := newTestService(t)
	ctx := context.Background()

	result, err := svc.Register(ctx, "bad-email", "password123", "Test User")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestRegister_SettingsCreateError(t *testing.T) {
	svc, users, tokens, settings, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
		u.ID = "user-456"
		return nil
	})
	settings.EXPECT().Upsert(ctx, gomock.Any()).Return(errors.New("db write failed"))
	// tokens should NOT be called since settings error happens first.
	_ = tokens

	result, err := svc.Register(ctx, "settings@example.com", "password123", "Test User")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create settings")
}

func TestRegister_TokenCreationError(t *testing.T) {
	svc, users, tokens, settings, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
		u.ID = "user-789"
		return nil
	})
	settings.EXPECT().Upsert(ctx, gomock.Any()).Return(nil)
	tokens.EXPECT().CreateRefreshToken(ctx, gomock.Any()).Return(errors.New("token store error"))

	result, err := svc.Register(ctx, "tokenerr@example.com", "password123", "Test User")
	assert.Nil(t, result)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Login — uncovered error paths
// ---------------------------------------------------------------------------

func TestLogin_GenericRepoError(t *testing.T) {
	svc, users, _, _, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().GetByEmail(ctx, "test@example.com").Return(nil, errors.New("db connection error"))

	result, err := svc.Login(ctx, "test@example.com", "password123", "", "")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection error")
}

func TestLogin_TokenCreationError(t *testing.T) {
	svc, users, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	users.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: string(hash),
	}, nil)
	tokens.EXPECT().CreateRefreshToken(ctx, gomock.Any()).Return(errors.New("token store error"))

	result, err := svc.Login(ctx, "test@example.com", "password123", "Mozilla/5.0", "127.0.0.1")
	assert.Nil(t, result)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Refresh — uncovered error paths
// ---------------------------------------------------------------------------

func TestRefresh_GetByHashError(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(nil, errors.New("db error"))

	result, err := svc.Refresh(ctx, "some-token")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestRefresh_RevokeByHashGenericError(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		UserID:    "user-123",
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	tokens.EXPECT().RevokeByHash(ctx, gomock.Any()).Return(errors.New("db write error"))

	result, err := svc.Refresh(ctx, "some-token")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db write error")
}

func TestRefresh_CreateRefreshTokenError(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		UserID:    "user-123",
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	tokens.EXPECT().RevokeByHash(ctx, gomock.Any()).Return(nil)
	tokens.EXPECT().CreateRefreshToken(ctx, gomock.Any()).Return(errors.New("token create error"))

	result, err := svc.Refresh(ctx, "some-token")
	assert.Nil(t, result)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Logout — uncovered error path
// ---------------------------------------------------------------------------

func TestLogout_RevokeError(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().RevokeByHash(ctx, gomock.Any()).Return(errors.New("db error"))

	err := svc.Logout(ctx, "some-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ---------------------------------------------------------------------------
// LogoutAll — uncovered error paths
// ---------------------------------------------------------------------------

func TestLogoutAll_RevokeAllError(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().RevokeAllForUser(ctx, "user-123").Return(errors.New("revoke all error"))

	err := svc.LogoutAll(ctx, "user-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoke all error")
}

func TestLogoutAll_BlacklistError(t *testing.T) {
	svc, _, tokens, _, blacklist := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().RevokeAllForUser(ctx, "user-123").Return(nil)
	blacklist.EXPECT().Add(ctx, "user_revoked:user-123", 15*time.Minute).Return(errors.New("blacklist error"))

	err := svc.LogoutAll(ctx, "user-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blacklist error")
}

// ---------------------------------------------------------------------------
// HashToken — basic test
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Register — bcrypt error path (via injectable bcryptGenerate)
// ---------------------------------------------------------------------------

func TestRegister_BcryptError(t *testing.T) {
	svc, _, _, _, _ := newTestService(t)
	ctx := context.Background()

	// Override bcryptGenerate to simulate failure.
	origBcrypt := usecase.BcryptGenerate
	usecase.BcryptGenerate = func(password []byte, cost int) ([]byte, error) {
		return nil, errors.New("bcrypt internal error")
	}
	defer func() { usecase.BcryptGenerate = origBcrypt }()

	result, err := svc.Register(ctx, "bcrypt@example.com", "password123", "Test User")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hash password")
}

// ---------------------------------------------------------------------------
// createRefreshToken — rand.Read error path (via injectable randReader)
// ---------------------------------------------------------------------------

func TestRegister_RandReadError(t *testing.T) {
	svc, users, _, settings, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
		u.ID = "user-rand"
		return nil
	})
	settings.EXPECT().Upsert(ctx, gomock.Any()).Return(nil)

	// Override randReader to simulate failure.
	origReader := usecase.RandReader
	usecase.RandReader = &failingReader{}
	defer func() { usecase.RandReader = origReader }()

	result, err := svc.Register(ctx, "rand@example.com", "password123", "Test User")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate refresh token")
}

// failingReader always returns an error.
type failingReader struct{}

func (r *failingReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("entropy source unavailable")
}

// ---------------------------------------------------------------------------
// generateAccessToken error — via broken JWT key
// ---------------------------------------------------------------------------

func TestRegister_JWTSigningError(t *testing.T) {
	svc, users, _, settings, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
		u.ID = "user-jwt"
		return nil
	})
	settings.EXPECT().Upsert(ctx, gomock.Any()).Return(nil)

	// Break JWT signing by setting a non-[]byte key.
	svc.SetJWTKey("not-a-byte-slice")

	result, err := svc.Register(ctx, "jwt@example.com", "password123", "Test User")
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestLogin_JWTSigningError(t *testing.T) {
	svc, users, _, _, _ := newTestService(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	users.EXPECT().GetByEmail(ctx, "jwt@example.com").Return(&domain.User{
		ID:           "user-jwt",
		Email:        "jwt@example.com",
		PasswordHash: string(hash),
	}, nil)

	svc.SetJWTKey("not-a-byte-slice")

	result, err := svc.Login(ctx, "jwt@example.com", "password123", "", "")
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestRefresh_JWTSigningError(t *testing.T) {
	svc, _, tokens, _, _ := newTestService(t)
	ctx := context.Background()

	tokens.EXPECT().GetByHash(ctx, gomock.Any()).Return(&domain.RefreshToken{
		UserID:    "user-jwt",
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)
	tokens.EXPECT().RevokeByHash(ctx, gomock.Any()).Return(nil)

	svc.SetJWTKey("not-a-byte-slice")

	result, err := svc.Refresh(ctx, "some-token")
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestHashToken(t *testing.T) {
	hash1 := usecase.HashToken("some-raw-token")
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA-256 hex = 64 chars

	// Deterministic: same input gives same output.
	hash2 := usecase.HashToken("some-raw-token")
	assert.Equal(t, hash1, hash2)

	// Different input gives different hash.
	hash3 := usecase.HashToken("different-token")
	assert.NotEqual(t, hash1, hash3)
}
