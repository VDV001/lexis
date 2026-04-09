package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/handler"
	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

// ---- mock repos ----

type mockWordRepo struct {
	words []domain.Word
}

func (m *mockWordRepo) Upsert(_ context.Context, w *domain.Word) error {
	m.words = append(m.words, *w)
	return nil
}
func (m *mockWordRepo) GetByUserAndWord(_ context.Context, _, _, _ string) (*domain.Word, error) {
	return nil, domain.ErrNotFound
}
func (m *mockWordRepo) ListByUser(_ context.Context, _, _ string, limit, _ int) ([]domain.Word, error) {
	if limit > len(m.words) {
		limit = len(m.words)
	}
	return m.words[:limit], nil
}
func (m *mockWordRepo) CountByStatus(_ context.Context, _, _ string) (int, int, int, int, error) {
	return len(m.words), 0, 0, 0, nil
}
func (m *mockWordRepo) GetDueForReview(_ context.Context, _, _ string, _ int) ([]domain.Word, error) {
	return m.words, nil
}
func (m *mockWordRepo) Delete(_ context.Context, id, _ string) error {
	for i, w := range m.words {
		if w.ID == id {
			m.words = append(m.words[:i], m.words[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}
func (m *mockWordRepo) UpdateStatus(_ context.Context, id, _ string, status domain.VocabStatus) error {
	for i, w := range m.words {
		if w.ID == id {
			m.words[i].Status = status
			return nil
		}
	}
	return domain.ErrNotFound
}
func (m *mockWordRepo) UpsertBatch(_ context.Context, words []*domain.Word) error {
	for _, w := range words {
		m.words = append(m.words, *w)
	}
	return nil
}
func (m *mockWordRepo) ListDistinctUserLanguages(_ context.Context) ([]domain.UserLanguage, error) {
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

func newHandler(words *mockWordRepo) *handler.VocabHandler {
	svc := usecase.NewVocabService(words, &mockSettingsRepo{})
	return handler.NewVocabHandler(svc)
}

func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

// ---- tests ----

func TestListWords_Success(t *testing.T) {
	repo := &mockWordRepo{words: []domain.Word{
		{ID: "1", Word: "hello", Language: "en", Status: domain.StatusUnknown},
	}}
	h := newHandler(repo)

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var words []domain.Word
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &words))
	assert.Len(t, words, 1)
	assert.Equal(t, "hello", words[0].Word)
}

func TestListWords_NoAuth(t *testing.T) {
	h := newHandler(&mockWordRepo{})

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAddWord_Success(t *testing.T) {
	repo := &mockWordRepo{}
	h := newHandler(repo)

	body, _ := json.Marshal(map[string]string{"word": "test", "status": "unknown"})
	r := httptest.NewRequestWithContext(context.Background(),"POST", "/", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Len(t, repo.words, 1)
}

func TestAddWord_InvalidStatus(t *testing.T) {
	h := newHandler(&mockWordRepo{})

	body, _ := json.Marshal(map[string]string{"word": "test", "status": "invalid"})
	r := httptest.NewRequestWithContext(context.Background(),"POST", "/", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddWord_EmptyWord(t *testing.T) {
	h := newHandler(&mockWordRepo{})

	body, _ := json.Marshal(map[string]string{"word": ""})
	r := httptest.NewRequestWithContext(context.Background(),"POST", "/", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteWord_Success(t *testing.T) {
	repo := &mockWordRepo{words: []domain.Word{
		{ID: "word-1", Word: "hello"},
	}}
	h := newHandler(repo)

	r := httptest.NewRequestWithContext(context.Background(),"DELETE", "/word-1", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	// Use chi router to parse URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "word-1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, repo.words)
}

func TestDeleteWord_NotFound(t *testing.T) {
	h := newHandler(&mockWordRepo{})

	r := httptest.NewRequestWithContext(context.Background(),"DELETE", "/nonexistent", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateWord_Success(t *testing.T) {
	repo := &mockWordRepo{words: []domain.Word{
		{ID: "word-1", Word: "hello", Status: domain.StatusUnknown},
	}}
	h := newHandler(repo)

	body, _ := json.Marshal(map[string]string{"status": "confident"})
	r := httptest.NewRequestWithContext(context.Background(),"PATCH", "/word-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, domain.StatusConfident, repo.words[0].Status)
}

func TestGetDueForReview_Success(t *testing.T) {
	repo := &mockWordRepo{words: []domain.Word{
		{ID: "1", Word: "hello"},
	}}
	h := newHandler(repo)

	r := httptest.NewRequestWithContext(context.Background(),"GET", "/due", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
