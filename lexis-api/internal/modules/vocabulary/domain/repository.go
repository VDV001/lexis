package domain

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("word not found")

type UserLanguage struct {
	UserID   string
	Language string
}

type WordRepository interface {
	Upsert(ctx context.Context, word *Word) error
	GetByUserAndWord(ctx context.Context, userID, word, language string) (*Word, error)
	ListByUser(ctx context.Context, userID, language string, limit, offset int) ([]Word, error)
	CountByStatus(ctx context.Context, userID, language string) (total, confident, uncertain, unknown int, err error)
	GetDueForReview(ctx context.Context, userID, language string, limit int) ([]Word, error)
	Delete(ctx context.Context, id, userID string) error
	UpdateStatus(ctx context.Context, id, userID string, status VocabStatus) error
	ListDistinctUserLanguages(ctx context.Context) ([]UserLanguage, error)
}

type SnapshotRepository interface {
	Create(ctx context.Context, snapshot *DailySnapshot) error
	GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]DailySnapshot, error)
}
