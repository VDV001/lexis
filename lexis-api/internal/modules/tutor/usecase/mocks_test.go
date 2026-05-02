package usecase_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/eventbus"
)

// noopPublisher discards all events (used when tests don't care about events).
type noopPublisher struct{}

func (noopPublisher) Publish(eventbus.Event) {}

// capturingPublisher records published events for assertions.
type capturingPublisher struct {
	events []eventbus.Event
}

func (p *capturingPublisher) Publish(e eventbus.Event) {
	p.events = append(p.events, e)
}

// mockRegistry implements usecase.ProviderRegistry for testing
type mockRegistry struct {
	providers map[string]usecase.AIProvider
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{providers: make(map[string]usecase.AIProvider)}
}

func (r *mockRegistry) Register(modelID string, p usecase.AIProvider) {
	r.providers[modelID] = p
}

func (r *mockRegistry) Get(modelID string) (usecase.AIProvider, error) {
	p, ok := r.providers[modelID]
	if !ok {
		return nil, fmt.Errorf("unknown model: %s", modelID)
	}
	return p, nil
}

// mockProvider implements usecase.AIProvider for testing
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

// mockSettingsRepo implements usecase.SettingsReader for testing.
type mockSettingsRepo struct{}

func (m *mockSettingsRepo) GetByUserID(_ context.Context, _ string) (*usecase.UserSettingsView, error) {
	return &usecase.UserSettingsView{
		TargetLanguage:   "en",
		ProficiencyLevel: "b1",
		VocabularyType:   "tech",
		AIModel:          "test-model",
	}, nil
}

// mockUserRepo implements usecase.UserReader for testing.
type mockUserRepo struct{}

func (m *mockUserRepo) GetByID(_ context.Context, _ string) (*usecase.UserView, error) {
	return &usecase.UserView{DisplayName: "Test User"}, nil
}

// Error-returning mocks for testing error paths

type mockSettingsRepoErr struct{}

func (m *mockSettingsRepoErr) GetByUserID(_ context.Context, _ string) (*usecase.UserSettingsView, error) {
	return nil, errors.New("settings db error")
}

type mockUserRepoErr struct{}

func (m *mockUserRepoErr) GetByID(_ context.Context, _ string) (*usecase.UserView, error) {
	return nil, errors.New("user db error")
}
