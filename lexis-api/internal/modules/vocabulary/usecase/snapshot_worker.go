package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

type VocabSnapshotWorker struct {
	wordRepo SnapshotWordReader
	snapRepo SnapshotRepository
}

func NewVocabSnapshotWorker(wordRepo SnapshotWordReader, snapRepo SnapshotRepository) *VocabSnapshotWorker {
	return &VocabSnapshotWorker{wordRepo: wordRepo, snapRepo: snapRepo}
}

// Run starts a daily timer that creates vocabulary snapshots at midnight UTC.
// It blocks until ctx is cancelled.
func (w *VocabSnapshotWorker) Run(ctx context.Context) {
	// Run once at startup
	if err := w.createSnapshots(ctx); err != nil {
		log.Printf("snapshot worker initial run: %v", err)
	}

	// Calculate duration until next midnight UTC to avoid ticker drift.
	now := time.Now().UTC()
	nextMidnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	timer := time.NewTimer(nextMidnight.Sub(now))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := w.createSnapshots(ctx); err != nil {
				log.Printf("snapshot worker: %v", err)
			}
			// Reset timer to next midnight UTC.
			now := time.Now().UTC()
			next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
			timer.Reset(next.Sub(now))
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
			log.Printf("snapshot worker: count for %s/%s: %v", ul.UserID, ul.Language, err)
			continue
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
			log.Printf("snapshot worker: create snapshot for %s/%s: %v", ul.UserID, ul.Language, err)
			continue
		}
		created++
	}

	log.Printf("snapshot worker: created %d/%d snapshots", created, len(pairs))
	return nil
}
