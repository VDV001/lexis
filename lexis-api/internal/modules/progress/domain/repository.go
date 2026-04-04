package domain

import (
	"context"
	"errors"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id, userID string) (*Session, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]Session, error)
	Update(ctx context.Context, session *Session) error
	IncrementCounters(ctx context.Context, id string, correct bool) error
}

type RoundRepository interface {
	Create(ctx context.Context, round *Round) error
	CountByUser(ctx context.Context, userID string) (total, correct int, err error)
	GetStreak(ctx context.Context, userID string) (int, error)
	GetErrorCounts(ctx context.Context, userID string) ([]ErrorCategory, error)
}

type GoalRepository interface {
	ListByUser(ctx context.Context, userID string) ([]Goal, error)
	CreateDefaults(ctx context.Context, userID, language string) error
	UpdateBatch(ctx context.Context, goals []Goal) error
}
