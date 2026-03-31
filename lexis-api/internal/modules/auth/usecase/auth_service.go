package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

type AuthService struct {
	users      domain.UserRepository
	tokens     domain.TokenRepository
	settings   domain.SettingsRepository
	blacklist  domain.Blacklist
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(
	users domain.UserRepository,
	tokens domain.TokenRepository,
	settings domain.SettingsRepository,
	blacklist domain.Blacklist,
	jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:      users,
		tokens:     tokens,
		settings:   settings,
		blacklist:  blacklist,
		jwtSecret:  []byte(jwtSecret),
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
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
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

func (s *AuthService) generateAccessToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) createRefreshToken(ctx context.Context, userID, userAgent, ip string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
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
