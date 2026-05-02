package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/lexis-app/lexis-api/internal/shared/eventbus"
)

type ExerciseService struct {
	registry ProviderRegistry
	settings SettingsReader
	bus      eventbus.Publisher
}

func NewExerciseService(registry ProviderRegistry, settings SettingsReader, bus eventbus.Publisher) *ExerciseService {
	return &ExerciseService{registry: registry, settings: settings, bus: bus}
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
		Mode:      input.Mode,
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

	result, err := provider.CheckAnswer(ctx, domain.CheckRequest{
		Mode:       input.Mode,
		System:     fmt.Sprintf("Check the user's answer for this %s exercise. Respond ONLY raw JSON.", input.Mode),
		Model:      settings.AIModel,
		UserAnswer: input.UserAnswer,
		Context:    input.Context,
		MaxTokens:  1024,
	})
	if err != nil {
		return domain.CheckResult{}, err
	}

	s.publishCheckEvents(input, result)

	return result, nil
}

// publishCheckEvents parses the AI check result and publishes domain events.
func (s *ExerciseService) publishCheckEvents(input CheckInput, result domain.CheckResult) {
	var parsed struct {
		Correct  bool     `json:"correct"`
		Word     string   `json:"word"`
		NewWords []string `json:"new_words"`
	}
	if err := json.Unmarshal([]byte(result.Raw), &parsed); err != nil {
		log.Printf("tutor: failed to parse check answer result: %v", err)
		return
	}

	s.bus.Publish(eventbus.Event{
		Type: eventbus.EventRoundCompleted,
		Payload: eventbus.RoundCompletedPayload{
			UserID:     input.UserID,
			Mode:       string(input.Mode),
			IsCorrect:  parsed.Correct,
			Question:   input.Context,
			UserAnswer: input.UserAnswer,
		},
	})

	var words []string
	if len(parsed.NewWords) > 0 {
		words = parsed.NewWords
	} else if parsed.Word != "" {
		words = []string{parsed.Word}
	}

	if len(words) > 0 {
		var exerciseCtx struct {
			Language string `json:"language"`
		}
		_ = json.Unmarshal([]byte(input.Context), &exerciseCtx)

		if exerciseCtx.Language != "" {
			s.bus.Publish(eventbus.Event{
				Type: eventbus.EventWordsDiscovered,
				Payload: eventbus.WordsDiscoveredPayload{
					UserID:   input.UserID,
					Language: exerciseCtx.Language,
					Words:    words,
					Context:  input.Context,
				},
			})
		}
	}
}
