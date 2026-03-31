package domain

import (
	"context"
	"time"
)

type WordRepository interface {
	Upsert(ctx context.Context, word *Word) error
	GetByUserAndWord(ctx context.Context, userID, word, language string) (*Word, error)
	ListByUser(ctx context.Context, userID, language string) ([]Word, error)
	CountByStatus(ctx context.Context, userID, language string) (total, confident, uncertain, unknown int, err error)
	GetDueForReview(ctx context.Context, userID, language string, limit int) ([]Word, error)
}

type SnapshotRepository interface {
	Create(ctx context.Context, snapshot *DailySnapshot) error
	GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]DailySnapshot, error)
}
