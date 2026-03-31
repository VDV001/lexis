package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

const TaskVocabSnapshot = "vocab:daily_snapshot"

type VocabSnapshotWorker struct {
	db       *pgxpool.Pool
	wordRepo domain.WordRepository
	snapRepo domain.SnapshotRepository
}

func NewVocabSnapshotWorker(db *pgxpool.Pool, wordRepo domain.WordRepository, snapRepo domain.SnapshotRepository) *VocabSnapshotWorker {
	return &VocabSnapshotWorker{db: db, wordRepo: wordRepo, snapRepo: snapRepo}
}

// NewSnapshotTask creates a new asynq task for the daily snapshot.
func NewSnapshotTask() *asynq.Task {
	return asynq.NewTask(TaskVocabSnapshot, nil)
}

// ProcessTask handles the daily snapshot job.
func (w *VocabSnapshotWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	// Get all active users with vocabulary
	rows, err := w.db.Query(ctx, `
		SELECT DISTINCT user_id, language
		FROM vocabulary_words
		WHERE user_id IN (SELECT id FROM users WHERE deleted_at IS NULL)
	`)
	if err != nil {
		return fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	type userLang struct {
		UserID   string
		Language string
	}

	var pairs []userLang
	for rows.Next() {
		var ul userLang
		if err := rows.Scan(&ul.UserID, &ul.Language); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		pairs = append(pairs, ul)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for _, ul := range pairs {
		total, confident, uncertain, unknown, err := w.wordRepo.CountByStatus(ctx, ul.UserID, ul.Language)
		if err != nil {
			return fmt.Errorf("count for %s/%s: %w", ul.UserID, ul.Language, err)
		}

		snapshot := &domain.DailySnapshot{
			UserID:       ul.UserID,
			Language:     ul.Language,
			SnapshotDate: today,
			TotalWords:   total,
			Confident:    confident,
			Uncertain:    uncertain,
			Unknown:      unknown,
		}

		if err := w.snapRepo.Create(ctx, snapshot); err != nil {
			return fmt.Errorf("create snapshot for %s/%s: %w", ul.UserID, ul.Language, err)
		}
	}

	return nil
}

