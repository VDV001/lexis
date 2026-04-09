package usecase_test

import (
	"context"
	"fmt"
	"time"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
)

// mockRegistry implements domain.ProviderRegistry for testing
type mockRegistry struct {
	providers map[string]domain.AIProvider
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{providers: make(map[string]domain.AIProvider)}
}

func (r *mockRegistry) Register(modelID string, p domain.AIProvider) {
	r.providers[modelID] = p
}

func (r *mockRegistry) Get(modelID string) (domain.AIProvider, error) {
	p, ok := r.providers[modelID]
	if !ok {
		return nil, fmt.Errorf("unknown model: %s", modelID)
	}
	return p, nil
}

// mockProvider implements domain.AIProvider for testing
type mockProvider struct{}

func (m *mockProvider) Chat(_ context.Context, _ domain.ChatRequest) (<-chan domain.ChatDelta, error) {
	ch := make(chan domain.ChatDelta, 5)
	go func() {
		ch <- domain.ChatDelta{Type: "delta", Content: "Hello! "}
		ch <- domain.ChatDelta{Type: "delta", Content: "How are you?"}
		ch <- domain.ChatDelta{Type: "done"}
		close(ch)
	}()
	return ch, nil
}

func (m *mockProvider) GenerateExercise(_ context.Context, _ domain.ExerciseRequest) (domain.Exercise, error) {
	return domain.Exercise{Raw: `{"question":"test","options":["a","b"],"correct":0}`}, nil
}

func (m *mockProvider) CheckAnswer(_ context.Context, _ domain.CheckRequest) (domain.CheckResult, error) {
	return domain.CheckResult{Raw: `{"correct":true,"explanation":"Good job"}`}, nil
}

// mockSettingsRepo implements authDomain.SettingsRepository for testing
type mockSettingsRepo struct{}

func (m *mockSettingsRepo) GetByUserID(_ context.Context, _ string) (*authDomain.UserSettings, error) {
	return &authDomain.UserSettings{
		UserID:           "user-123",
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

// mockUserRepo implements authDomain.UserRepository for testing
type mockUserRepo struct{}

func (m *mockUserRepo) Create(_ context.Context, _ *authDomain.User) error { return nil }
func (m *mockUserRepo) GetByID(_ context.Context, id string) (*authDomain.User, error) {
	return &authDomain.User{ID: id, DisplayName: "Test User"}, nil
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*authDomain.User, error) {
	return nil, authDomain.ErrUserNotFound
}
func (m *mockUserRepo) Update(_ context.Context, _ *authDomain.User) error { return nil }
