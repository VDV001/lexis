package usecase

import (
	"context"
	"time"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	progressDomain "github.com/lexis-app/lexis-api/internal/modules/progress/domain"
	vocabDomain "github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

type SessionRepository interface {
	Create(ctx context.Context, session *progressDomain.Session) error
	GetByID(ctx context.Context, id, userID string) (*progressDomain.Session, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]progressDomain.Session, error)
	Update(ctx context.Context, session *progressDomain.Session) error
	IncrementCounters(ctx context.Context, id string, correct bool) error
}

type RoundRepository interface {
	Create(ctx context.Context, round *progressDomain.Round) error
	CountByUser(ctx context.Context, userID string) (total, correct int, err error)
	GetStreak(ctx context.Context, userID string) (int, error)
	GetErrorCounts(ctx context.Context, userID string) ([]progressDomain.ErrorCategory, error)
}

type GoalRepository interface {
	ListByUser(ctx context.Context, userID string) ([]progressDomain.Goal, error)
	CreateDefaults(ctx context.Context, userID, language string) error
	UpdateBatch(ctx context.Context, goals []progressDomain.Goal) error
}

type WordCounter interface {
	CountByStatus(ctx context.Context, userID, language string) (total, confident, uncertain, unknown int, err error)
}

type SnapshotReader interface {
	GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]vocabDomain.DailySnapshot, error)
}

type SettingsReader interface {
	GetByUserID(ctx context.Context, userID string) (*authDomain.UserSettings, error)
}
