package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

// --- Internal mocks (package-level access) ---

type stubWordReader struct {
	pairs    []domain.UserLanguage
	counts   map[string][4]int
	pairsErr error
	countErr map[string]error
}

func (s *stubWordReader) ListDistinctUserLanguages(_ context.Context) ([]domain.UserLanguage, error) {
	return s.pairs, s.pairsErr
}

func (s *stubWordReader) CountByStatus(_ context.Context, userID, language string) (int, int, int, int, error) {
	key := userID + "/" + language
	if s.countErr != nil {
		if err, ok := s.countErr[key]; ok {
			return 0, 0, 0, 0, err
		}
	}
	if c, ok := s.counts[key]; ok {
		return c[0], c[1], c[2], c[3], nil
	}
	return 0, 0, 0, 0, nil
}

type stubSnapRepo struct {
	created []*domain.DailySnapshot
	err     error
}

func (s *stubSnapRepo) Create(_ context.Context, snap *domain.DailySnapshot) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, snap)
	return nil
}

func (s *stubSnapRepo) GetByDateRange(_ context.Context, _, _ string, _, _ time.Time) ([]domain.DailySnapshot, error) {
	return nil, nil
}

// --- Tests for createSnapshots ---

func TestCreateSnapshots_CancelledContext(t *testing.T) {
	w := &VocabSnapshotWorker{
		wordRepo: &stubWordReader{
			pairs: []domain.UserLanguage{{UserID: "u1", Language: "en"}},
		},
		snapRepo: &stubSnapRepo{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := w.createSnapshots(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestCreateSnapshots_ListDistinctError(t *testing.T) {
	w := &VocabSnapshotWorker{
		wordRepo: &stubWordReader{pairsErr: errors.New("db down")},
		snapRepo: &stubSnapRepo{},
	}

	err := w.createSnapshots(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list user languages")
}

func TestCreateSnapshots_CountByStatusError(t *testing.T) {
	snaps := &stubSnapRepo{}
	w := &VocabSnapshotWorker{
		wordRepo: &stubWordReader{
			pairs: []domain.UserLanguage{
				{UserID: "u1", Language: "en"},
				{UserID: "u2", Language: "de"},
			},
			counts: map[string][4]int{
				"u2/de": {5, 2, 2, 1},
			},
			countErr: map[string]error{
				"u1/en": errors.New("count fail"),
			},
		},
		snapRepo: snaps,
	}

	err := w.createSnapshots(context.Background())
	assert.NoError(t, err) // Errors are logged per-pair, not returned.
	assert.Len(t, snaps.created, 1)
	assert.Equal(t, "u2", snaps.created[0].UserID)
}

func TestCreateSnapshots_CreateError(t *testing.T) {
	snaps := &stubSnapRepo{err: errors.New("create fail")}
	w := &VocabSnapshotWorker{
		wordRepo: &stubWordReader{
			pairs:  []domain.UserLanguage{{UserID: "u1", Language: "en"}},
			counts: map[string][4]int{"u1/en": {10, 5, 3, 2}},
		},
		snapRepo: snaps,
	}

	err := w.createSnapshots(context.Background())
	assert.NoError(t, err) // Create errors are logged per-pair, not returned.
	assert.Empty(t, snaps.created)
}

// --- Tests for Run ---

func TestRun_InitialErrorIsLogged(t *testing.T) {
	snaps := &stubSnapRepo{}
	w := &VocabSnapshotWorker{
		wordRepo:     &stubWordReader{pairsErr: errors.New("startup fail")},
		snapRepo:     snaps,
		nextInterval: untilMidnightUTC,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit")
	}

	assert.Empty(t, snaps.created)
}

func TestRun_TimerFiresAndProcesses(t *testing.T) {
	snaps := &stubSnapRepo{}
	callCount := 0
	w := &VocabSnapshotWorker{
		wordRepo: &stubWordReader{
			pairs:  []domain.UserLanguage{{UserID: "u1", Language: "en"}},
			counts: map[string][4]int{"u1/en": {10, 5, 3, 2}},
		},
		snapRepo: snaps,
		nextInterval: func() time.Duration {
			callCount++
			return 5 * time.Millisecond // fire almost immediately
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	// Wait for initial run + at least one timer fire.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit")
	}

	// Initial run creates 1 snapshot, timer fires at least once creating more.
	assert.GreaterOrEqual(t, len(snaps.created), 2)
	assert.GreaterOrEqual(t, callCount, 2) // initial + at least 1 reset
}

func TestRun_TimerFiresWithError(t *testing.T) {
	callCount := 0
	w := &VocabSnapshotWorker{
		wordRepo: &stubWordReader{pairsErr: errors.New("always fails")},
		snapRepo: &stubSnapRepo{},
		nextInterval: func() time.Duration {
			callCount++
			return 5 * time.Millisecond
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit")
	}

	// nextInterval was called at least twice (initial timer + reset after error).
	assert.GreaterOrEqual(t, callCount, 2)
}
