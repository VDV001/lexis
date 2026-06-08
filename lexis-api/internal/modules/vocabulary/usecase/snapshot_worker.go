package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

type VocabSnapshotWorker struct {
	wordRepo SnapshotWordReader
	snapRepo SnapshotRepository

	// nextInterval returns the duration until the next tick. Defaults to
	// midnight-UTC calculation. Overridable for tests.
	nextInterval func() time.Duration
}

func NewVocabSnapshotWorker(wordRepo SnapshotWordReader, snapRepo SnapshotRepository) *VocabSnapshotWorker {
	return &VocabSnapshotWorker{
		wordRepo: wordRepo,
		snapRepo: snapRepo,
		nextInterval: untilMidnightUTC,
	}
}

func untilMidnightUTC() time.Duration {
	now := time.Now().UTC()
	return now.Truncate(24 * time.Hour).Add(24 * time.Hour).Sub(now)
}

// Run starts a daily timer that creates vocabulary snapshots at midnight UTC.
// It blocks until ctx is cancelled.
func (w *VocabSnapshotWorker) Run(ctx context.Context) {
	// Run once at startup
	if err := w.createSnapshots(ctx); err != nil {
		slog.Error("snapshot worker initial run", "error", err)
	}

	// Calculate duration until next tick to avoid ticker drift.
	timer := time.NewTimer(w.nextInterval())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := w.createSnapshots(ctx); err != nil {
				slog.Error("snapshot worker error", "error", err)
			}
			timer.Reset(w.nextInterval())
		}
	}
}

func (w *VocabSnapshotWorker) createSnapshots(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	pairs, err := w.wordRepo.ListDistinctUserLanguages(ctx)
	if err != nil {
		return fmt.Errorf("list user languages: %w", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	var created int
	for _, ul := range pairs {
		total, confident, uncertain, unknown, err := w.wordRepo.CountByStatus(ctx, ul.UserID, ul.Language)
		if err != nil {
			slog.Error("snapshot worker: count failed", "user_id", ul.UserID, "language", ul.Language, "error", err)
			continue
		}

		snapshot := domain.NewDailySnapshot(ul.UserID, ul.Language, today, total, confident, uncertain, unknown)

		if err := w.snapRepo.Create(ctx, snapshot); err != nil {
			slog.Error("snapshot worker: create snapshot failed", "user_id", ul.UserID, "language", ul.Language, "error", err)
			continue
		}
		created++
	}

	slog.Info("snapshot worker: created snapshots", "created", created, "total", len(pairs))
	return nil
}
