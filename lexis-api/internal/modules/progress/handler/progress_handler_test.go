package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// ---- failing mocks ----

var errMock = errors.New("mock error")

type failRoundRepo struct{}

func (m *failRoundRepo) Create(_ context.Context, _ *progressDomain.Round) error { return errMock }
func (m *failRoundRepo) CountByUser(_ context.Context, _ string) (int, int, error) {
	return 0, 0, errMock
}
func (m *failRoundRepo) GetStreak(_ context.Context, _ string) (int, error) { return 0, errMock }
func (m *failRoundRepo) GetErrorCounts(_ context.Context, _ string) ([]progressDomain.ErrorCategory, error) {
	return nil, errMock
}

type failSessionRepo struct{}

func (m *failSessionRepo) Create(_ context.Context, _ *progressDomain.Session) error { return errMock }
func (m *failSessionRepo) GetByID(_ context.Context, _, _ string) (*progressDomain.Session, error) {
	return nil, errMock
}
func (m *failSessionRepo) ListByUser(_ context.Context, _ string, _, _ int) ([]progressDomain.Session, error) {
	return nil, errMock
}
func (m *failSessionRepo) Update(_ context.Context, _ *progressDomain.Session) error { return errMock }
func (m *failSessionRepo) IncrementCounters(_ context.Context, _ string, _ bool) error {
	return errMock
}

type failGoalRepo struct{}

func (m *failGoalRepo) ListByUser(_ context.Context, _ string) ([]progressDomain.Goal, error) {
	return nil, errMock
}
func (m *failGoalRepo) CreateDefaults(_ context.Context, _, _ string) error { return errMock }
func (m *failGoalRepo) UpdateBatch(_ context.Context, _ []progressDomain.Goal) error { return errMock }

type failWordRepo struct{}

func (m *failWordRepo) Upsert(_ context.Context, _ *vocabDomain.Word) error { return errMock }
func (m *failWordRepo) GetByUserAndWord(_ context.Context, _, _, _ string) (*vocabDomain.Word, error) {
	return nil, errMock
}
func (m *failWordRepo) ListByUser(_ context.Context, _, _ string, _, _ int) ([]vocabDomain.Word, error) {
	return nil, errMock
}
func (m *failWordRepo) CountByStatus(_ context.Context, _, _ string) (int, int, int, int, error) {
	return 0, 0, 0, 0, errMock
}
func (m *failWordRepo) GetDueForReview(_ context.Context, _, _ string, _ int) ([]vocabDomain.Word, error) {
	return nil, errMock
}
func (m *failWordRepo) Delete(_ context.Context, _, _ string) error { return errMock }
func (m *failWordRepo) UpdateStatus(_ context.Context, _, _ string, _ vocabDomain.VocabStatus) error {
	return errMock
}
func (m *failWordRepo) UpsertBatch(_ context.Context, _ []*vocabDomain.Word) error { return errMock }
func (m *failWordRepo) ListDistinctUserLanguages(_ context.Context) ([]vocabDomain.UserLanguage, error) {
	return nil, errMock
}

type failSnapshotRepo struct{}

func (m *failSnapshotRepo) Create(_ context.Context, _ *vocabDomain.DailySnapshot) error {
	return errMock
}
func (m *failSnapshotRepo) GetByDateRange(_ context.Context, _, _ string, _, _ time.Time) ([]vocabDomain.DailySnapshot, error) {
	return nil, errMock
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

func newFailingService() *usecase.ProgressService {
	return usecase.NewProgressService(
		&failRoundRepo{},
		&failSessionRepo{},
		&failGoalRepo{},
		&failWordRepo{},
		&failSnapshotRepo{},
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

// ---- Additional coverage tests ----

func TestHandleVocabulary_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/vocabulary", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleVocabulary_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/vocabulary", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleVocabCurve_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/vocabulary/curve", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleVocabCurve_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/vocabulary/curve", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleGoals_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/goals", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleErrors_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/errors", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleSessions_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleSessions_WithOffset(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions?limit=5&offset=10", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleSessions_LimitCapped(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions?limit=500", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleSessions_InvalidLimitOffset(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions?limit=abc&offset=xyz", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code) // defaults used
}

func TestHandleSession_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions/sess-1", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var session progressDomain.Session
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &session))
	assert.Equal(t, "sess-1", session.ID)
}

func TestHandleSession_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions/sess-1", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleStartSession_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	body, _ := json.Marshal(map[string]string{
		"mode": "quiz", "language": "en", "level": "b1", "ai_model": "test",
	})
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/sessions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleStartSession_InvalidJSON(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "POST", "/sessions", bytes.NewReader([]byte("not json")))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleStartSession_InvalidMode(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	body, _ := json.Marshal(map[string]string{
		"mode": "invalid_mode", "language": "en", "level": "b1", "ai_model": "test",
	})
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/sessions", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleRecordRound_Success(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	body, _ := json.Marshal(map[string]interface{}{
		"session_id":  "sess-1",
		"mode":        "quiz",
		"is_correct":  true,
		"question":    "What is 2+2?",
		"user_answer": "4",
	})
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/rounds", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandleRecordRound_NoAuth(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	body, _ := json.Marshal(map[string]interface{}{
		"session_id":  "sess-1",
		"mode":        "quiz",
		"is_correct":  true,
		"question":    "q",
		"user_answer": "a",
	})
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/rounds", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleRecordRound_InvalidJSON(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "POST", "/rounds", bytes.NewReader([]byte("bad")))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleRecordRound_SessionNotFound(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	body, _ := json.Marshal(map[string]interface{}{
		"session_id":  "nonexistent",
		"mode":        "quiz",
		"is_correct":  true,
		"question":    "q",
		"user_answer": "a",
	})
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/rounds", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSessions_DefaultParams(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleSessions_NegativeOffset(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions?offset=-5", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code) // negative offset is ignored, defaults to 0
}

func TestHandleSessions_ZeroLimit(t *testing.T) {
	h := handler.NewProgressHandler(newService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions?limit=0", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code) // zero limit ignored, defaults to 20
}

// ---- Service error path tests ----

func TestHandleSummary_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/summary", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleVocabulary_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/vocabulary", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleVocabCurve_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/vocabulary/curve", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleGoals_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/goals", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleErrors_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/errors", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleSessions_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleSession_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	r := httptest.NewRequestWithContext(context.Background(), "GET", "/sessions/some-id", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleStartSession_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	body, _ := json.Marshal(map[string]string{
		"mode": "quiz", "language": "en", "level": "b1", "ai_model": "test",
	})
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/sessions", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleRecordRound_ServiceError(t *testing.T) {
	h := handler.NewProgressHandler(newFailingService())

	body, _ := json.Marshal(map[string]interface{}{
		"session_id":  "sess-1",
		"mode":        "quiz",
		"is_correct":  true,
		"question":    "q",
		"user_answer": "a",
	})
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/rounds", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
