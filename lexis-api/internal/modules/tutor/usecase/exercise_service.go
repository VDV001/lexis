package usecase

import (
	"context"
	"fmt"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/infra"
)

type ExerciseService struct {
	registry *infra.ProviderRegistry
	settings authDomain.SettingsRepository
}

func NewExerciseService(registry *infra.ProviderRegistry, settings authDomain.SettingsRepository) *ExerciseService {
	return &ExerciseService{registry: registry, settings: settings}
}

type GenerateInput struct {
	UserID string
	Mode   domain.Mode
}

func (s *ExerciseService) Generate(ctx context.Context, input GenerateInput) (domain.Exercise, error) {
	settings, err := s.settings.GetByUserID(ctx, input.UserID)
	if err != nil {
		return domain.Exercise{}, fmt.Errorf("get settings: %w", err)
	}

	provider, err := s.registry.Get(settings.AIModel)
	if err != nil {
		return domain.Exercise{}, fmt.Errorf("get provider: %w", err)
	}

	promptSettings := domain.PromptSettings{
		TargetLanguage:   settings.TargetLanguage,
		ProficiencyLevel: settings.ProficiencyLevel,
		VocabularyType:   settings.VocabularyType,
	}

	systemPrompt := domain.BuildSystemPrompt(promptSettings, input.Mode)

	return provider.GenerateExercise(ctx, domain.ExerciseRequest{
		Mode:      string(input.Mode),
		System:    systemPrompt,
		Model:     settings.AIModel,
		MaxTokens: 1024,
	})
}

type CheckInput struct {
	UserID     string
	Mode       domain.Mode
	UserAnswer string
	Context    string // Original exercise JSON
}

func (s *ExerciseService) Check(ctx context.Context, input CheckInput) (domain.CheckResult, error) {
	settings, err := s.settings.GetByUserID(ctx, input.UserID)
	if err != nil {
		return domain.CheckResult{}, fmt.Errorf("get settings: %w", err)
	}

	provider, err := s.registry.Get(settings.AIModel)
	if err != nil {
		return domain.CheckResult{}, fmt.Errorf("get provider: %w", err)
	}

	return provider.CheckAnswer(ctx, domain.CheckRequest{
		Mode:       string(input.Mode),
		System:     fmt.Sprintf("Check the user's answer for this %s exercise. Respond ONLY raw JSON.", input.Mode),
		Model:      settings.AIModel,
		UserAnswer: input.UserAnswer,
		Context:    input.Context,
		MaxTokens:  1024,
	})
}
