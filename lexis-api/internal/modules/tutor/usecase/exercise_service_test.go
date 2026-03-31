package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/infra"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
)

func TestExerciseService_Generate(t *testing.T) {
	registry := infra.NewProviderRegistry()
	registry.Register("test-model", &mockProvider{})

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{})

	exercise, err := svc.Generate(context.Background(), usecase.GenerateInput{
		UserID: "user-123",
		Mode:   domain.ModeQuiz,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, exercise.Raw)
}

func TestExerciseService_Check(t *testing.T) {
	registry := infra.NewProviderRegistry()
	registry.Register("test-model", &mockProvider{})

	svc := usecase.NewExerciseService(registry, &mockSettingsRepo{})

	result, err := svc.Check(context.Background(), usecase.CheckInput{
		UserID:     "user-123",
		Mode:       domain.ModeQuiz,
		UserAnswer: "rides",
		Context:    `{"question":"...","correct":0}`,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Raw)
}
