package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authdomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
)

// --- Mocks ---

type mockWordRepo struct {
	upserted     []*domain.Word
	deleted      []string
	updatedPairs []statusUpdate
	listResult   []domain.Word
	dueResult    []domain.Word
	err          error
}

type statusUpdate struct {
	ID     string
	UserID string
	Status domain.VocabStatus
}

func (m *mockWordRepo) Upsert(ctx context.Context, w *domain.Word) error {
	if m.err != nil {
		return m.err
	}
	m.upserted = append(m.upserted, w)
	return nil
}

func (m *mockWordRepo) GetByUserAndWord(ctx context.Context, userID, word, language string) (*domain.Word, error) {
	return nil, nil
}

func (m *mockWordRepo) ListByUser(ctx context.Context, userID, language string, limit, offset int) ([]domain.Word, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.listResult, nil
}

func (m *mockWordRepo) CountByStatus(ctx context.Context, userID, language string) (int, int, int, int, error) {
	return 0, 0, 0, 0, nil
}

func (m *mockWordRepo) GetDueForReview(ctx context.Context, userID, language string, limit int) ([]domain.Word, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.dueResult, nil
}

func (m *mockWordRepo) Delete(ctx context.Context, id, userID string) error {
	if m.err != nil {
		return m.err
	}
	m.deleted = append(m.deleted, id)
	return nil
}

func (m *mockWordRepo) UpdateStatus(ctx context.Context, id, userID string, status domain.VocabStatus) error {
	if m.err != nil {
		return m.err
	}
	m.updatedPairs = append(m.updatedPairs, statusUpdate{ID: id, UserID: userID, Status: status})
	return nil
}

func (m *mockWordRepo) UpsertBatch(_ context.Context, words []*domain.Word) error {
	if m.err != nil {
		return m.err
	}
	m.upserted = append(m.upserted, words...)
	return nil
}
func (m *mockWordRepo) ListDistinctUserLanguages(_ context.Context) ([]domain.UserLanguage, error) {
	return nil, nil
}

type mockSettingsRepo struct {
	settings *authdomain.UserSettings
	err      error
}

func (m *mockSettingsRepo) GetByUserID(ctx context.Context, userID string) (*authdomain.UserSettings, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.settings, nil
}

func (m *mockSettingsRepo) Upsert(ctx context.Context, s *authdomain.UserSettings) error {
	return nil
}

// --- Tests ---

func TestAddWord_WithExplicitLanguage(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	word, err := svc.AddWord(context.Background(), usecase.AddWordInput{
		UserID:   "u1",
		Word:     "hello",
		Language: "de",
		Status:   domain.StatusUncertain,
		Context:  "Hallo!",
	})

	require.NoError(t, err)
	assert.Equal(t, "de", word.Language)
	assert.Equal(t, domain.StatusUncertain, word.Status)
	assert.Equal(t, "hello", word.Word)
	assert.Equal(t, 2.5, word.EaseFactor)
	assert.NotEmpty(t, word.ID)
	assert.Len(t, words.upserted, 1)
}

func TestAddWord_FallsBackToUserLanguage(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "fr"}}
	svc := usecase.NewVocabService(words, settings)

	word, err := svc.AddWord(context.Background(), usecase.AddWordInput{
		UserID: "u1",
		Word:   "bonjour",
	})

	require.NoError(t, err)
	assert.Equal(t, "fr", word.Language)
	assert.Equal(t, domain.StatusUnknown, word.Status)
}

func TestAddWord_InvalidStatus(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	_, err := svc.AddWord(context.Background(), usecase.AddWordInput{
		UserID:   "u1",
		Word:     "test",
		Language: "en",
		Status:   "garbage",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, usecase.ErrInvalidStatus))
	assert.Empty(t, words.upserted)
}

func TestAddDiscoveredWords_UpsertsBatch(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.AddDiscoveredWords(context.Background(), "u1", "en", []string{"hello", "world"}, "lesson 1")

	require.NoError(t, err)
	require.Len(t, words.upserted, 2)
	assert.Equal(t, "hello", words.upserted[0].Word)
	assert.Equal(t, "world", words.upserted[1].Word)
	assert.Equal(t, domain.StatusUnknown, words.upserted[0].Status)
	assert.Equal(t, "lesson 1", words.upserted[0].Context)
}

func TestAddDiscoveredWords_StopsOnError(t *testing.T) {
	words := &mockWordRepo{err: errors.New("db error")}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.AddDiscoveredWords(context.Background(), "u1", "en", []string{"hello"}, "")

	require.Error(t, err)
}

func TestUpdateStatus_Valid(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.UpdateStatus(context.Background(), "w1", "u1", domain.StatusConfident)

	require.NoError(t, err)
	require.Len(t, words.updatedPairs, 1)
	assert.Equal(t, domain.StatusConfident, words.updatedPairs[0].Status)
}

func TestUpdateStatus_InvalidStatus(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.UpdateStatus(context.Background(), "w1", "u1", "nope")

	require.Error(t, err)
	assert.True(t, errors.Is(err, usecase.ErrInvalidStatus))
	assert.Empty(t, words.updatedPairs)
}

func TestListWords_UsesSettingsLanguage(t *testing.T) {
	expected := []domain.Word{{ID: "1", Word: "test"}}
	words := &mockWordRepo{listResult: expected}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "es"}}
	svc := usecase.NewVocabService(words, settings)

	result, err := svc.ListWords(context.Background(), "u1", 500, 0)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestDeleteWord(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.DeleteWord(context.Background(), "w1", "u1")

	require.NoError(t, err)
	assert.Equal(t, []string{"w1"}, words.deleted)
}

func TestGetDueForReview(t *testing.T) {
	due := []domain.Word{{ID: "1", Word: "review-me", NextReview: time.Now().Add(-time.Hour)}}
	words := &mockWordRepo{dueResult: due}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	result, err := svc.GetDueForReview(context.Background(), "u1", 50)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "review-me", result[0].Word)
}

// ---- Error path tests ----

func TestAddWord_SettingsError(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{err: errors.New("settings db down")}
	svc := usecase.NewVocabService(words, settings)

	_, err := svc.AddWord(context.Background(), usecase.AddWordInput{
		UserID: "u1",
		Word:   "hello",
		// Language intentionally empty to trigger settings lookup
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "settings db down")
	assert.Empty(t, words.upserted)
}

func TestAddWord_UpsertError(t *testing.T) {
	words := &mockWordRepo{err: errors.New("upsert failed")}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	_, err := svc.AddWord(context.Background(), usecase.AddWordInput{
		UserID:   "u1",
		Word:     "hello",
		Language: "en",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert failed")
}

func TestAddWord_EmptyUserID(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	_, err := svc.AddWord(context.Background(), usecase.AddWordInput{
		UserID:   "",
		Word:     "hello",
		Language: "en",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserRequired)
}

func TestAddWord_WithNonDefaultStatus(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	word, err := svc.AddWord(context.Background(), usecase.AddWordInput{
		UserID:   "u1",
		Word:     "hello",
		Language: "en",
		Status:   domain.StatusConfident,
	})

	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfident, word.Status)
}

func TestAddWord_DefaultStatusUnknown(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	word, err := svc.AddWord(context.Background(), usecase.AddWordInput{
		UserID:   "u1",
		Word:     "hello",
		Language: "en",
		// Status intentionally empty
	})

	require.NoError(t, err)
	assert.Equal(t, domain.StatusUnknown, word.Status)
}

func TestListWords_SettingsError(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{err: errors.New("settings fail")}
	svc := usecase.NewVocabService(words, settings)

	_, err := svc.ListWords(context.Background(), "u1", 500, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "settings fail")
}

func TestListWords_RepoError(t *testing.T) {
	words := &mockWordRepo{err: errors.New("list fail")}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	_, err := svc.ListWords(context.Background(), "u1", 500, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list fail")
}

func TestDeleteWord_RepoError(t *testing.T) {
	words := &mockWordRepo{err: errors.New("delete fail")}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.DeleteWord(context.Background(), "w1", "u1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete fail")
}

func TestUpdateStatus_RepoError(t *testing.T) {
	words := &mockWordRepo{err: errors.New("update fail")}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.UpdateStatus(context.Background(), "w1", "u1", domain.StatusConfident)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "update fail")
}

func TestGetDueForReview_SettingsError(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{err: errors.New("settings fail")}
	svc := usecase.NewVocabService(words, settings)

	_, err := svc.GetDueForReview(context.Background(), "u1", 50)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "settings fail")
}

func TestGetDueForReview_RepoError(t *testing.T) {
	words := &mockWordRepo{err: errors.New("due fail")}
	settings := &mockSettingsRepo{settings: &authdomain.UserSettings{TargetLanguage: "en"}}
	svc := usecase.NewVocabService(words, settings)

	_, err := svc.GetDueForReview(context.Background(), "u1", 50)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "due fail")
}

func TestAddDiscoveredWords_EmptyWord(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.AddDiscoveredWords(context.Background(), "u1", "en", []string{""}, "ctx")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrWordRequired)
}

func TestAddDiscoveredWords_EmptyUserID(t *testing.T) {
	words := &mockWordRepo{}
	settings := &mockSettingsRepo{}
	svc := usecase.NewVocabService(words, settings)

	err := svc.AddDiscoveredWords(context.Background(), "", "en", []string{"hello"}, "ctx")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserRequired)
}
