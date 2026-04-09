package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	progressDomain "github.com/lexis-app/lexis-api/internal/modules/progress/domain"
	"github.com/lexis-app/lexis-api/internal/modules/progress/handler"
	"github.com/lexis-app/lexis-api/internal/modules/progress/usecase"
	vocabDomain "github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

// ---- mock repos ----

type mockRoundRepo struct{}

func (m *mockRoundRepo) Create(_ context.Context, _ *progressDomain.Round) error { return nil }
func (m *mockRoundRepo) CountByUser(_ context.Context, _ string) (int, int, error) {
	return 10, 7, nil
}
func (m *mockRoundRepo) GetStreak(_ context.Context, _ string) (int, error) { return 3, nil }
func (m *mockRoundRepo) GetErrorCounts(_ context.Context, _ string) ([]progressDomain.ErrorCategory, error) {
	return []progressDomain.ErrorCategory{{ErrorType: "articles", Count: 2}}, nil
}

type mockSessionRepo struct {
	sessions []progressDomain.Session
}

func (m *mockSessionRepo) Create(_ context.Context, s *progressDomain.Session) error {
	m.sessions = append(m.sessions, *s)
	return nil
}
func (m *mockSessionRepo) GetByID(_ context.Context, id, userID string) (*progressDomain.Session, error) {
	for _, s := range m.sessions {
		if s.ID == id && s.UserID == userID {
			return &s, nil
		}
	}
	return nil, progressDomain.ErrSessionNotFound
}
func (m *mockSessionRepo) ListByUser(_ context.Context, _ string, _, _ int) ([]progressDomain.Session, error) {
	return m.sessions, nil
}
func (m *mockSessionRepo) Update(_ context.Context, _ *progressDomain.Session) error { return nil }
func (m *mockSessionRepo) IncrementCounters(_ context.Context, _ string, _ bool) error {
	return nil
}

type mockGoalRepo struct{}

func (m *mockGoalRepo) ListByUser(_ context.Context, _ string) ([]progressDomain.Goal, error) {
	return []progressDomain.Goal{
		{ID: "g1", Name: "Accuracy", Progress: 50, Color: "green"},
	}, nil
}
func (m *mockGoalRepo) CreateDefaults(_ context.Context, _, _ string) error { return nil }
func (m *mockGoalRepo) UpdateBatch(_ context.Context, _ []progressDomain.Goal) error {
	return nil
}

type mockWordRepo struct{}

func (m *mockWordRepo) Upsert(_ context.Context, _ *vocabDomain.Word) error { return nil }
func (m *mockWordRepo) GetByUserAndWord(_ context.Context, _, _, _ string) (*vocabDomain.Word, error) {
	return nil, vocabDomain.ErrNotFound
}
func (m *mockWordRepo) ListByUser(_ context.Context, _, _ string, _, _ int) ([]vocabDomain.Word, error) {
	return nil, nil
}
func (m *mockWordRepo) CountByStatus(_ context.Context, _, _ string) (int, int, int, int, error) {
	return 100, 30, 40, 30, nil
}
func (m *mockWordRepo) GetDueForReview(_ context.Context, _, _ string, _ int) ([]vocabDomain.Word, error) {
	return nil, nil
}
func (m *mockWordRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockWordRepo) UpdateStatus(_ context.Context, _, _ string, _ vocabDomain.VocabStatus) error {
	return nil
}
func (m *mockWordRepo) UpsertBatch(_ context.Context, _ []*vocabDomain.Word) error { return nil }
func (m *mockWordRepo) ListDistinctUserLanguages(_ context.Context) ([]vocabDomain.UserLanguage, error) {
	return nil, nil
}

type mockSnapshotRepo struct{}

func (m *mockSnapshotRepo) Create(_ context.Context, _ *vocabDomain.DailySnapshot) error {
	return nil
}
func (m *mockSnapshotRepo) GetByDateRange(_ context.Context, _, _ string, _, _ time.Time) ([]vocabDomain.DailySnapshot, error) {
	return nil, nil
}

type mockSettingsRepo struct{}

func (m *mockSettingsRepo) GetByUserID(_ context.Context, _ string) (*authDomain.UserSettings, error) {
	return &authDomain.UserSettings{
		TargetLanguage:   "en",
		ProficiencyLevel: "b1",
		VocabularyType:   "tech",
		AIModel:          "test-model",
		VocabGoal:        3000,
		UILanguage:       "ru",
		UpdatedAt:        time.Now(),
	}, nil
}
func (m *mockSettingsRepo) Upsert(_ context.Context, _ *authDomain.UserSettings) error {
	return nil
}

// ---- helpers ----

func newService() *usecase.ProgressService {
	return usecase.NewProgressService(
		&mockRoundRepo{},
		&mockSessionRepo{sessions: []progressDomain.Session{
			{ID: "sess-1", UserID: "user-1", Mode: "quiz", Language: "en", Level: "b1", AIModel: "test"},
		}},
		&mockGoalRepo{},
		&mockWordRepo{},
		&mockSnapshotRepo{},
		&mockSettingsRepo{},
	)
}

func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

// ---- tests ----

func TestHandleSummary_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/summary", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var body progressDomain.ProgressSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 10, body.TotalRounds)
	assert.Equal(t, 7, body.CorrectRounds)
	assert.Equal(t, 3, body.Streak)
}

func TestHandleSummary_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/summary", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleGoals_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/goals", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var goals []progressDomain.Goal
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &goals))
	assert.Len(t, goals, 1)
}

func TestHandleErrors_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/errors", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleSessions_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/sessions?limit=10", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleSession_NotFound(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/sessions/nonexistent", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleStartSession_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	body, _ := json.Marshal(map[string]string{
		"mode": "quiz", "language": "en", "level": "b1", "ai_model": "test",
	})
	r := httptest.NewRequestWithContext(context.Background(),"POST", "/sessions", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["id"])
}

func TestHandleStartSession_MissingFields(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	body, _ := json.Marshal(map[string]string{"mode": "quiz"})
	r := httptest.NewRequestWithContext(context.Background(),"POST", "/sessions", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleRecordRound_MissingFields(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	body, _ := json.Marshal(map[string]string{"session_id": "sess-1"})
	r := httptest.NewRequestWithContext(context.Background(),"POST", "/rounds", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
