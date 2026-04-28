package usecase

import (
	"context"
	"time"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

type WordRepository interface {
	Upsert(ctx context.Context, word *domain.Word) error
	UpsertBatch(ctx context.Context, words []*domain.Word) error
	GetByUserAndWord(ctx context.Context, userID, word, language string) (*domain.Word, error)
	ListByUser(ctx context.Context, userID, language string, limit, offset int) ([]domain.Word, error)
	CountByStatus(ctx context.Context, userID, language string) (total, confident, uncertain, unknown int, err error)
	GetDueForReview(ctx context.Context, userID, language string, limit int) ([]domain.Word, error)
	Delete(ctx context.Context, id, userID string) error
	UpdateStatus(ctx context.Context, id, userID string, status domain.VocabStatus) error
	ListDistinctUserLanguages(ctx context.Context) ([]domain.UserLanguage, error)
}

type SettingsReader interface {
	GetByUserID(ctx context.Context, userID string) (*authDomain.UserSettings, error)
}

type SnapshotWordReader interface {
	CountByStatus(ctx context.Context, userID, language string) (total, confident, uncertain, unknown int, err error)
	ListDistinctUserLanguages(ctx context.Context) ([]domain.UserLanguage, error)
}

type SnapshotRepository interface {
	Create(ctx context.Context, snapshot *domain.DailySnapshot) error
	GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]domain.DailySnapshot, error)
}
