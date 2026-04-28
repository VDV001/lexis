package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/eventbus"
)

func TestExerciseService_Generate(t *testing.T) {
	registry := newMockRegistry()
	registry.Register("test-model", &mockProvider{})

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, &noopPublisher{})

	exercise, err := svc.Generate(context.Background(), usecase.GenerateInput{
		UserID: "user-123",
		Mode:   domain.ModeQuiz,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, exercise.Raw)
}

func TestExerciseService_Check(t *testing.T) {
	registry := newMockRegistry()
	registry.Register("test-model", &mockProvider{})

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, &noopPublisher{})

	result, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "rides",
		Context:    `{"question":"...","correct":0}`,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Raw)
}

func TestExerciseService_Generate_SettingsError(t *testing.T) {
	registry := newMockRegistry()
	registry.Register("test-model", &mockProvider{})

	svc := usecase.NewExerciseService(registry, &mockSettingsRepoErr{}, &noopPublisher{})

	_, err := svc.Generate(context.Background(), usecase.GenerateInput{
		UserID: "user-123",
		Mode:   domain.ModeQuiz,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get settings")
}

func TestExerciseService_Generate_UnknownModel(t *testing.T) {
	registry := newMockRegistry() // no providers registered

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, &noopPublisher{})

	_, err := svc.Generate(context.Background(), usecase.GenerateInput{
		UserID: "user-123",
		Mode:   domain.ModeQuiz,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get provider")
}

func TestExerciseService_Check_SettingsError(t *testing.T) {
	registry := newMockRegistry()
	registry.Register("test-model", &mockProvider{})

	svc := usecase.NewExerciseService(registry, &mockSettingsRepoErr{}, &noopPublisher{})

	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "test",
		Context:    `{}`,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get settings")
}

func TestExerciseService_Check_UnknownModel(t *testing.T) {
	registry := newMockRegistry() // no providers registered

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, &noopPublisher{})

	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "test",
		Context:    `{}`,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get provider")
}

// ---------------------------------------------------------------------------
// Event publishing tests (moved from handler layer)
// ---------------------------------------------------------------------------

func newCheckProvider(raw string) *eventCheckProvider {
	return &eventCheckProvider{raw: raw}
}

type eventCheckProvider struct {
	mockProvider
	raw string
}

func (p *eventCheckProvider) CheckAnswer(_ context.Context, _ domain.CheckRequest) (domain.CheckResult, error) {
	return domain.CheckResult{Raw: p.raw}, nil
}

func TestExerciseService_Check_PublishesRoundCompleted(t *testing.T) {
	registry := newMockRegistry()
	prov := newCheckProvider(`{"correct":true,"word":"hello"}`)
	registry.Register("test-model", prov)
	pub := &capturingPublisher{}

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, pub)

	ctxJSON := `{"question":"translate hello","language":"en"}`
	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "hola",
		Context:    ctxJSON,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(pub.events), 1)

	assert.Equal(t, eventbus.EventRoundCompleted, pub.events[0].Type)
	payload, ok := pub.events[0].Payload.(eventbus.RoundCompletedPayload)
	require.True(t, ok)
	assert.Equal(t, "user-123", payload.UserID)
	assert.Equal(t, "quiz", payload.Mode)
	assert.True(t, payload.IsCorrect)
	assert.Equal(t, ctxJSON, payload.Question)
	assert.Equal(t, "hola", payload.UserAnswer)
}

func TestExerciseService_Check_SingleWordPublishesWordsDiscovered(t *testing.T) {
	registry := newMockRegistry()
	prov := newCheckProvider(`{"correct":true,"word":"bonjour"}`)
	registry.Register("test-model", prov)
	pub := &capturingPublisher{}

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, pub)

	ctxJSON := `{"language":"en","question":"say hello"}`
	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "bonjour",
		Context:    ctxJSON,
	})
	require.NoError(t, err)
	require.Len(t, pub.events, 2)

	assert.Equal(t, eventbus.EventWordsDiscovered, pub.events[1].Type)
	wp, ok := pub.events[1].Payload.(eventbus.WordsDiscoveredPayload)
	require.True(t, ok)
	assert.Equal(t, "user-123", wp.UserID)
	assert.Equal(t, "en", wp.Language)
	assert.Equal(t, []string{"bonjour"}, wp.Words)
	assert.Equal(t, ctxJSON, wp.Context)
}

func TestExerciseService_Check_NewWordsPublishesWordsDiscovered(t *testing.T) {
	registry := newMockRegistry()
	prov := newCheckProvider(`{"correct":false,"new_words":["apple","banana","cherry"]}`)
	registry.Register("test-model", prov)
	pub := &capturingPublisher{}

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, pub)

	ctxJSON := `{"language":"en","question":"name fruits"}`
	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "wrong",
		Context:    ctxJSON,
	})
	require.NoError(t, err)
	require.Len(t, pub.events, 2)

	assert.Equal(t, eventbus.EventWordsDiscovered, pub.events[1].Type)
	wp, ok := pub.events[1].Payload.(eventbus.WordsDiscoveredPayload)
	require.True(t, ok)
	assert.Equal(t, []string{"apple", "banana", "cherry"}, wp.Words)
}

func TestExerciseService_Check_NewWordsTakesPrecedenceOverWord(t *testing.T) {
	registry := newMockRegistry()
	prov := newCheckProvider(`{"correct":true,"word":"single","new_words":["multi1","multi2"]}`)
	registry.Register("test-model", prov)
	pub := &capturingPublisher{}

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, pub)

	ctxJSON := `{"language":"en"}`
	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "answer",
		Context:    ctxJSON,
	})
	require.NoError(t, err)
	require.Len(t, pub.events, 2)

	wp, ok := pub.events[1].Payload.(eventbus.WordsDiscoveredPayload)
	require.True(t, ok)
	assert.Equal(t, []string{"multi1", "multi2"}, wp.Words)
}

func TestExerciseService_Check_NoLanguage_NoWordsDiscoveredEvent(t *testing.T) {
	registry := newMockRegistry()
	prov := newCheckProvider(`{"correct":true,"word":"hello"}`)
	registry.Register("test-model", prov)
	pub := &capturingPublisher{}

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, pub)

	ctxJSON := `{"question":"say hello"}`
	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "hello",
		Context:    ctxJSON,
	})
	require.NoError(t, err)
	require.Len(t, pub.events, 1)
	assert.Equal(t, eventbus.EventRoundCompleted, pub.events[0].Type)
}

func TestExerciseService_Check_NoWords_NoWordsDiscoveredEvent(t *testing.T) {
	registry := newMockRegistry()
	prov := newCheckProvider(`{"correct":false}`)
	registry.Register("test-model", prov)
	pub := &capturingPublisher{}

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, pub)

	ctxJSON := `{"language":"en"}`
	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "wrong",
		Context:    ctxJSON,
	})
	require.NoError(t, err)
	require.Len(t, pub.events, 1)
}

func TestExerciseService_Check_UnparsableResult_NoEvents(t *testing.T) {
	registry := newMockRegistry()
	// json.Unmarshal into struct will fail for a bare number
	prov := newCheckProvider(`42`)
	registry.Register("test-model", prov)
	pub := &capturingPublisher{}

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{}, pub)

	_, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "x",
		Context:    `{"language":"en"}`,
	})
	require.NoError(t, err)
	assert.Empty(t, pub.events)
}
