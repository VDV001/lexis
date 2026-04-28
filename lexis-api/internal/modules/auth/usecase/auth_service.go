package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

// Package-level variables for crypto operations, replaceable in tests.
var (
	RandReader     io.Reader = rand.Reader
	BcryptGenerate            = bcrypt.GenerateFromPassword
)

type AuthService struct {
	users      UserRepository
	tokens     TokenRepository
	settings   SettingsRepository
	blacklist  Blacklist
	jwtKey     any // normally []byte; typed as any so SignedString can surface errors
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(
	users UserRepository,
	tokens TokenRepository,
	settings SettingsRepository,
	blacklist Blacklist,
	jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:      users,
		tokens:     tokens,
		settings:   settings,
		blacklist:  blacklist,
		jwtKey:     []byte(jwtSecret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

type AuthResult struct {
	User         *domain.User
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*AuthResult, error) {
	if err := domain.ValidatePassword(password); err != nil {
		return nil, err
	}

	hash, err := BcryptGenerate([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(email, string(hash), displayName)
	if err != nil {
		return nil, err
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	// Create default settings
	defaults := domain.DefaultSettings(user.ID)
	if err := s.settings.Upsert(ctx, &defaults); err != nil {
		return nil, fmt.Errorf("create settings: %w", err)
	}

	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := s.createRefreshToken(ctx, user.ID, "", "")
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password, userAgent, ip string) (*AuthResult, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := s.createRefreshToken(ctx, user.ID, userAgent, ip)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

// TokenResult holds the new access/refresh token pair returned by Refresh.
type TokenResult struct {
	AccessToken  string
	RefreshToken string
}

// Refresh validates the given raw refresh token, revokes it (rotation),
// and returns a new access + refresh token pair.
//
// Token reuse detection: if a revoked token is presented, all tokens for that
// user are revoked (token family revocation). The atomic RevokeByHash call
// (WHERE revoked_at IS NULL) acts as the concurrency guard — only one of two
// concurrent callers can succeed; the loser sees ErrTokenNotFound which is
// treated as reuse.
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*TokenResult, error) {
	hash := sha256Hash(rawRefreshToken)

	token, err := s.tokens.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	if token.IsExpired(time.Now()) {
		return nil, domain.ErrTokenExpired
	}

	// If token is already revoked, this is a reuse attack — revoke ALL tokens for user.
	if token.IsRevoked() {
		_ = s.tokens.RevokeAllForUser(ctx, token.UserID)
		return nil, domain.ErrTokenRevoked
	}

	// Atomically revoke — if RevokeByHash returns ErrTokenNotFound,
	// another request already revoked it (race condition = reuse).
	if err := s.tokens.RevokeByHash(ctx, hash); err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			_ = s.tokens.RevokeAllForUser(ctx, token.UserID)
			return nil, domain.ErrTokenRevoked
		}
		return nil, err
	}

	accessToken, err := s.generateAccessToken(token.UserID)
	if err != nil {
		return nil, err
	}

	newRawRefresh, err := s.createRefreshToken(ctx, token.UserID, "", "")
	if err != nil {
		return nil, err
	}

	return &TokenResult{
		AccessToken:  accessToken,
		RefreshToken: newRawRefresh,
	}, nil
}

// Logout revokes a single refresh token.
// Access tokens are short-lived (15 min) and expire naturally.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := sha256Hash(rawRefreshToken)

	if err := s.tokens.RevokeByHash(ctx, hash); err != nil {
		return err
	}

	return nil
}

// LogoutAll revokes all refresh tokens for a given user and blacklists
// the user ID so that existing access tokens are immediately rejected
// by the auth middleware (TTL = access token lifetime).
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	if err := s.tokens.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	return s.blacklist.Add(ctx, "user_revoked:"+userID, s.accessTTL)
}

// HashToken exposes SHA-256 hashing of a raw token string for use by handlers.
func HashToken(raw string) string {
	return sha256Hash(raw)
}

func (s *AuthService) generateAccessToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtKey)
}

func (s *AuthService) createRefreshToken(ctx context.Context, userID, userAgent, ip string) (string, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(RandReader, raw); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	rawHex := hex.EncodeToString(raw)
	hash := sha256Hash(rawHex)

	token := &domain.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
		UserAgent: userAgent,
		IPAddress: ip,
	}

	if err := s.tokens.CreateRefreshToken(ctx, token); err != nil {
		return "", err
	}

	return rawHex, nil
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
