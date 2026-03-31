package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
}

type TokenRepository interface {
	CreateRefreshToken(ctx context.Context, token *RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeByHash(ctx context.Context, hash string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

type SettingsRepository interface {
	GetByUserID(ctx context.Context, userID string) (*UserSettings, error)
	Upsert(ctx context.Context, settings *UserSettings) error
}

type Blacklist interface {
	Add(ctx context.Context, tokenHash string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, tokenHash string) (bool, error)
}
