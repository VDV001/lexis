package usecase

import (
	"context"
	"time"

	progressDomain "github.com/lexis-app/lexis-api/internal/modules/progress/domain"
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

// DailySnapshotView is the projection of a vocabulary daily snapshot the
// progress module exposes in its API responses. The vocabulary module owns
// the source type; an adapter in main.go projects it to this view.
type DailySnapshotView struct {
	UserID       string    `json:"user_id"`
	Language     string    `json:"language"`
	SnapshotDate time.Time `json:"date"`
	TotalWords   int       `json:"total"`
	Confident    int       `json:"confident"`
	Uncertain    int       `json:"uncertain"`
	Unknown      int       `json:"unknown"`
}

type SnapshotReader interface {
	GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]DailySnapshotView, error)
}

// UserSettingsView is the narrow projection of user settings the progress
// module consumes (target language for vocab counts, vocab goal for the
// curve response). Adapter in main.go projects auth UserSettings to this.
type UserSettingsView struct {
	TargetLanguage string
	VocabGoal      int
}

type SettingsReader interface {
	GetByUserID(ctx context.Context, userID string) (*UserSettingsView, error)
}
