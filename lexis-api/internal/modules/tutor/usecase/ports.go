package usecase

import (
	"context"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

type SettingsReader interface {
	GetByUserID(ctx context.Context, userID string) (*authDomain.UserSettings, error)
}

type UserReader interface {
	GetByID(ctx context.Context, id string) (*authDomain.User, error)
}
