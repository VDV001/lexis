package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
)

type mockSnapshotRepo struct {
	created []*domain.DailySnapshot
	err     error
}

func (m *mockSnapshotRepo) Create(ctx context.Context, s *domain.DailySnapshot) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, s)
	return nil
}

func (m *mockSnapshotRepo) GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]domain.DailySnapshot, error) {
	return nil, nil
}

// mockWordRepoForWorker extends mockWordRepo with snapshot-specific behavior.
type mockWordRepoForWorker struct {
	mockWordRepo
	userLanguages []domain.UserLanguage
	statusCounts  map[string][4]int // key: "userID/language" → [total, confident, uncertain, unknown]
}

func (m *mockWordRepoForWorker) ListDistinctUserLanguages(ctx context.Context) ([]domain.UserLanguage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.userLanguages, nil
}

func (m *mockWordRepoForWorker) CountByStatus(ctx context.Context, userID, language string) (int, int, int, int, error) {
	key := userID + "/" + language
	if c, ok := m.statusCounts[key]; ok {
		return c[0], c[1], c[2], c[3], nil
	}
	return 0, 0, 0, 0, nil
}

func TestNewVocabSnapshotWorker(t *testing.T) {
	worker := usecase.NewVocabSnapshotWorker(nil, nil)
	assert.NotNil(t, worker)
}

func TestSnapshotWorkerRunExitsOnCancelledContext(t *testing.T) {
	words := &mockWordRepoForWorker{}
	snaps := &mockSnapshotRepo{}
	worker := usecase.NewVocabSnapshotWorker(words, snaps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
}

func TestSnapshotWorkerCreatesSnapshots(t *testing.T) {
	words := &mockWordRepoForWorker{
		userLanguages: []domain.UserLanguage{
			{UserID: "u1", Language: "en"},
			{UserID: "u2", Language: "de"},
		},
		statusCounts: map[string][4]int{
			"u1/en": {10, 5, 3, 2},
			"u2/de": {7, 2, 4, 1},
		},
	}
	snaps := &mockSnapshotRepo{}
	worker := usecase.NewVocabSnapshotWorker(words, snaps)

	// Run with an immediately-cancelled context so Run does initial createSnapshots then exits.
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	// Give it a moment to run createSnapshots, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	require.Len(t, snaps.created, 2)

	assert.Equal(t, "u1", snaps.created[0].UserID)
	assert.Equal(t, "en", snaps.created[0].Language)
	assert.Equal(t, 10, snaps.created[0].TotalWords)
	assert.Equal(t, 5, snaps.created[0].Confident)

	assert.Equal(t, "u2", snaps.created[1].UserID)
	assert.Equal(t, "de", snaps.created[1].Language)
	assert.Equal(t, 7, snaps.created[1].TotalWords)
}
