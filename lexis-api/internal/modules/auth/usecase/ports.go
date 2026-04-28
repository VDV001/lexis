package usecase

import (
	"context"
	"time"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
}

type TokenRepository interface {
	CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	RevokeByHash(ctx context.Context, hash string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

type SettingsRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.UserSettings, error)
	Upsert(ctx context.Context, settings *domain.UserSettings) error
}

type Blacklist interface {
	Add(ctx context.Context, tokenHash string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, tokenHash string) (bool, error)
}
