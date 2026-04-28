package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	progressDomain "github.com/lexis-app/lexis-api/internal/modules/progress/domain"
	vocabDomain "github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

// --- Mocks ---

type mockRounds struct {
	total, correct int
	streak         int
	errors         []progressDomain.ErrorCategory
	createErr      error
	countErr       error
	streakErr      error
	errorsErr      error
}

func (m *mockRounds) Create(_ context.Context, _ *progressDomain.Round) error { return m.createErr }
func (m *mockRounds) CountByUser(_ context.Context, _ string) (int, int, error) {
	return m.total, m.correct, m.countErr
}
func (m *mockRounds) GetStreak(_ context.Context, _ string) (int, error) {
	return m.streak, m.streakErr
}
func (m *mockRounds) GetErrorCounts(_ context.Context, _ string) ([]progressDomain.ErrorCategory, error) {
	return m.errors, m.errorsErr
}

type mockSessions struct {
	session   *progressDomain.Session
	sessions  []progressDomain.Session
	createErr error
	getErr    error
	listErr   error
	incrErr   error
}

func (m *mockSessions) Create(_ context.Context, _ *progressDomain.Session) error { return m.createErr }
func (m *mockSessions) GetByID(_ context.Context, _, _ string) (*progressDomain.Session, error) {
	return m.session, m.getErr
}
func (m *mockSessions) ListByUser(_ context.Context, _ string, _, _ int) ([]progressDomain.Session, error) {
	return m.sessions, m.listErr
}
func (m *mockSessions) Update(_ context.Context, _ *progressDomain.Session) error { return nil }
func (m *mockSessions) IncrementCounters(_ context.Context, _ string, _ bool) error {
	return m.incrErr
}

type mockGoals struct {
	goals     []progressDomain.Goal
	listErr   error
	updateErr error
}

func (m *mockGoals) ListByUser(_ context.Context, _ string) ([]progressDomain.Goal, error) {
	return m.goals, m.listErr
}
func (m *mockGoals) CreateDefaults(_ context.Context, _, _ string) error { return nil }
func (m *mockGoals) UpdateBatch(_ context.Context, goals []progressDomain.Goal) error {
	m.goals = goals
	return m.updateErr
}

type mockWords struct {
	total, confident, uncertain, unknown int
	err                                  error
}

func (m *mockWords) CountByStatus(_ context.Context, _, _ string) (int, int, int, int, error) {
	return m.total, m.confident, m.uncertain, m.unknown, m.err
}

type mockSnaps struct {
	snapshots []vocabDomain.DailySnapshot
	err       error
}

func (m *mockSnaps) GetByDateRange(_ context.Context, _, _ string, _, _ time.Time) ([]vocabDomain.DailySnapshot, error) {
	return m.snapshots, m.err
}

type mockSettings struct {
	settings *authDomain.UserSettings
	err      error
}

func (m *mockSettings) GetByUserID(_ context.Context, _ string) (*authDomain.UserSettings, error) {
	return m.settings, m.err
}

func defaultSettings() *authDomain.UserSettings {
	s := authDomain.DefaultSettings("user-1")
	return &s
}

func newService(rounds *mockRounds, sessions *mockSessions, goals *mockGoals, words *mockWords, snaps *mockSnaps, settings *mockSettings) *ProgressService {
	return NewProgressService(rounds, sessions, goals, words, snaps, settings)
}

// --- GetSummary ---

func TestGetSummary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newService(
			&mockRounds{total: 10, correct: 7, streak: 3},
			&mockSessions{},
			&mockGoals{},
			&mockWords{total: 50},
			&mockSnaps{},
			&mockSettings{settings: defaultSettings()},
		)
		summary, err := svc.GetSummary(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, 10, summary.TotalRounds)
		assert.Equal(t, 7, summary.CorrectRounds)
		assert.Equal(t, 70.0, summary.Accuracy)
		assert.Equal(t, 3, summary.Streak)
		assert.Equal(t, 50, summary.TotalWords)
	})

	t.Run("zero rounds — accuracy is 0", func(t *testing.T) {
		svc := newService(
			&mockRounds{total: 0, correct: 0},
			&mockSessions{},
			&mockGoals{},
			&mockWords{},
			&mockSnaps{},
			&mockSettings{settings: defaultSettings()},
		)
		summary, err := svc.GetSummary(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, 0.0, summary.Accuracy)
	})

	t.Run("rounds error", func(t *testing.T) {
		svc := newService(
			&mockRounds{countErr: errors.New("db")},
			&mockSessions{},
			&mockGoals{},
			&mockWords{},
			&mockSnaps{},
			&mockSettings{settings: defaultSettings()},
		)
		_, err := svc.GetSummary(context.Background(), "user-1")
		assert.Error(t, err)
	})

	t.Run("streak error", func(t *testing.T) {
		svc := newService(
			&mockRounds{streakErr: errors.New("db")},
			&mockSessions{},
			&mockGoals{},
			&mockWords{},
			&mockSnaps{},
			&mockSettings{settings: defaultSettings()},
		)
		_, err := svc.GetSummary(context.Background(), "user-1")
		assert.Error(t, err)
	})

	t.Run("settings error", func(t *testing.T) {
		svc := newService(
			&mockRounds{total: 1, correct: 1},
			&mockSessions{},
			&mockGoals{},
			&mockWords{},
			&mockSnaps{},
			&mockSettings{err: errors.New("db")},
		)
		_, err := svc.GetSummary(context.Background(), "user-1")
		assert.Error(t, err)
	})

	t.Run("words error", func(t *testing.T) {
		svc := newService(
			&mockRounds{total: 1, correct: 1},
			&mockSessions{},
			&mockGoals{},
			&mockWords{err: errors.New("db")},
			&mockSnaps{},
			&mockSettings{settings: defaultSettings()},
		)
		_, err := svc.GetSummary(context.Background(), "user-1")
		assert.Error(t, err)
	})
}

// --- GetVocabulary ---

func TestGetVocabulary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newService(
			&mockRounds{},
			&mockSessions{},
			&mockGoals{},
			&mockWords{total: 100, confident: 50, uncertain: 30, unknown: 20},
			&mockSnaps{},
			&mockSettings{settings: defaultSettings()},
		)
		stats, err := svc.GetVocabulary(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, 100, stats.Total)
		assert.Equal(t, 50, stats.Confident)
	})

	t.Run("settings error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{}, &mockGoals{}, &mockWords{},
			&mockSnaps{}, &mockSettings{err: errors.New("db")},
		)
		_, err := svc.GetVocabulary(context.Background(), "user-1")
		assert.Error(t, err)
	})

	t.Run("words error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{}, &mockGoals{},
			&mockWords{err: errors.New("db")},
			&mockSnaps{}, &mockSettings{settings: defaultSettings()},
		)
		_, err := svc.GetVocabulary(context.Background(), "user-1")
		assert.Error(t, err)
	})
}

// --- GetVocabCurve ---

func TestGetVocabCurve(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		snap := vocabDomain.DailySnapshot{TotalWords: 42}
		svc := newService(
			&mockRounds{},
			&mockSessions{},
			&mockGoals{},
			&mockWords{total: 42},
			&mockSnaps{snapshots: []vocabDomain.DailySnapshot{snap}},
			&mockSettings{settings: defaultSettings()},
		)
		curve, err := svc.GetVocabCurve(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, 3000, curve.Goal)
		assert.Len(t, curve.Daily, 1)
	})

	t.Run("settings error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{}, &mockGoals{}, &mockWords{},
			&mockSnaps{}, &mockSettings{err: errors.New("db")},
		)
		_, err := svc.GetVocabCurve(context.Background(), "user-1")
		assert.Error(t, err)
	})

	t.Run("vocab error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{}, &mockGoals{},
			&mockWords{err: errors.New("db")},
			&mockSnaps{}, &mockSettings{settings: defaultSettings()},
		)
		_, err := svc.GetVocabCurve(context.Background(), "user-1")
		assert.Error(t, err)
	})

	t.Run("snaps error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{}, &mockGoals{}, &mockWords{},
			&mockSnaps{err: errors.New("db")}, &mockSettings{settings: defaultSettings()},
		)
		_, err := svc.GetVocabCurve(context.Background(), "user-1")
		assert.Error(t, err)
	})
}

// --- StartSession ---

func TestStartSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		id, err := svc.StartSession(context.Background(), "user-1", "chat", "en", "b1", "claude-sonnet")
		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("invalid mode", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		_, err := svc.StartSession(context.Background(), "user-1", "invalid", "en", "b1", "model")
		assert.ErrorIs(t, err, progressDomain.ErrInvalidMode)
	})

	t.Run("empty userID", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		_, err := svc.StartSession(context.Background(), "", "chat", "en", "b1", "model")
		assert.Error(t, err)
	})

	t.Run("create error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{createErr: errors.New("db")}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		_, err := svc.StartSession(context.Background(), "user-1", "chat", "en", "b1", "model")
		assert.Error(t, err)
	})
}

// --- RecordRound ---

func TestRecordRound(t *testing.T) {
	session := &progressDomain.Session{ID: "s-1", UserID: "user-1"}

	t.Run("success with goals", func(t *testing.T) {
		goals := &mockGoals{goals: []progressDomain.Goal{{Name: "g1", Progress: 50, Color: "amber"}}}
		svc := newService(
			&mockRounds{}, &mockSessions{session: session}, goals, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		err := svc.RecordRound(context.Background(), RecordRoundInput{
			SessionID: "s-1", UserID: "user-1", Mode: "chat",
			IsCorrect: true, Question: "q", UserAnswer: "a",
		})
		require.NoError(t, err)
		assert.Equal(t, 53, goals.goals[0].Progress)
	})

	t.Run("success without goals", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{session: session}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		err := svc.RecordRound(context.Background(), RecordRoundInput{
			SessionID: "s-1", UserID: "user-1", Mode: "chat",
			IsCorrect: false, Question: "q", UserAnswer: "a",
		})
		require.NoError(t, err)
	})

	t.Run("session not found", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{getErr: progressDomain.ErrSessionNotFound}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		err := svc.RecordRound(context.Background(), RecordRoundInput{
			SessionID: "bad", UserID: "user-1", Mode: "chat",
			IsCorrect: true, Question: "q", UserAnswer: "a",
		})
		assert.ErrorIs(t, err, progressDomain.ErrSessionNotFound)
	})

	t.Run("round create error", func(t *testing.T) {
		svc := newService(
			&mockRounds{createErr: errors.New("db")}, &mockSessions{session: session}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		err := svc.RecordRound(context.Background(), RecordRoundInput{
			SessionID: "s-1", UserID: "user-1", Mode: "chat",
			IsCorrect: true, Question: "q", UserAnswer: "a",
		})
		assert.Error(t, err)
	})

	t.Run("increment error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{session: session, incrErr: errors.New("db")}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		err := svc.RecordRound(context.Background(), RecordRoundInput{
			SessionID: "s-1", UserID: "user-1", Mode: "chat",
			IsCorrect: true, Question: "q", UserAnswer: "a",
		})
		assert.Error(t, err)
	})

	t.Run("goals list error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{session: session}, &mockGoals{listErr: errors.New("db")}, &mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		err := svc.RecordRound(context.Background(), RecordRoundInput{
			SessionID: "s-1", UserID: "user-1", Mode: "chat",
			IsCorrect: true, Question: "q", UserAnswer: "a",
		})
		assert.Error(t, err)
	})

	t.Run("goals update error", func(t *testing.T) {
		svc := newService(
			&mockRounds{}, &mockSessions{session: session},
			&mockGoals{goals: []progressDomain.Goal{{Name: "g1", Progress: 50}}, updateErr: errors.New("db")},
			&mockWords{}, &mockSnaps{}, &mockSettings{},
		)
		err := svc.RecordRound(context.Background(), RecordRoundInput{
			SessionID: "s-1", UserID: "user-1", Mode: "chat",
			IsCorrect: true, Question: "q", UserAnswer: "a",
		})
		assert.Error(t, err)
	})
}

// --- Passthrough methods ---

func TestGetGoals(t *testing.T) {
	goals := []progressDomain.Goal{{Name: "g1"}}
	svc := newService(&mockRounds{}, &mockSessions{}, &mockGoals{goals: goals}, &mockWords{}, &mockSnaps{}, &mockSettings{})
	result, err := svc.GetGoals(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestGetErrors(t *testing.T) {
	errs := []progressDomain.ErrorCategory{{ErrorType: "grammar", Count: 5}}
	svc := newService(&mockRounds{errors: errs}, &mockSessions{}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{})
	result, err := svc.GetErrors(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, 5, result[0].Count)
}

func TestGetSessions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		sessions := []progressDomain.Session{{ID: "s-1"}}
		svc := newService(&mockRounds{}, &mockSessions{sessions: sessions}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{})
		result, err := svc.GetSessions(context.Background(), "user-1", 10, 0)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("error", func(t *testing.T) {
		svc := newService(&mockRounds{}, &mockSessions{listErr: errors.New("db")}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{})
		_, err := svc.GetSessions(context.Background(), "user-1", 10, 0)
		assert.Error(t, err)
	})
}

func TestGetSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		session := &progressDomain.Session{ID: "s-1"}
		svc := newService(&mockRounds{}, &mockSessions{session: session}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{})
		result, err := svc.GetSession(context.Background(), "s-1", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "s-1", result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newService(&mockRounds{}, &mockSessions{getErr: progressDomain.ErrSessionNotFound}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{})
		_, err := svc.GetSession(context.Background(), "bad", "user-1")
		assert.ErrorIs(t, err, progressDomain.ErrSessionNotFound)
	})
}

func TestGetGoals_Error(t *testing.T) {
	svc := newService(&mockRounds{}, &mockSessions{}, &mockGoals{listErr: errors.New("db")}, &mockWords{}, &mockSnaps{}, &mockSettings{})
	_, err := svc.GetGoals(context.Background(), "user-1")
	assert.Error(t, err)
}

func TestGetErrors_Error(t *testing.T) {
	svc := newService(&mockRounds{errorsErr: errors.New("db")}, &mockSessions{}, &mockGoals{}, &mockWords{}, &mockSnaps{}, &mockSettings{})
	_, err := svc.GetErrors(context.Background(), "user-1")
	assert.Error(t, err)
}
