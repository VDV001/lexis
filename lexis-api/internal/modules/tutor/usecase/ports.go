package usecase

import (
	"context"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
)

// ProviderRegistry resolves a model ID to its AIProvider.
type ProviderRegistry interface {
	Get(modelID string) (AIProvider, error)
}

// AIProvider is the interface for all AI model providers.
type AIProvider interface {
	// Chat streams a response for free practice mode.
	Chat(ctx context.Context, req domain.ChatRequest) (<-chan domain.ChatDelta, error)

	// GenerateExercise generates a quiz/translate/gap/scramble exercise.
	GenerateExercise(ctx context.Context, req domain.ExerciseRequest) (domain.Exercise, error)

	// CheckAnswer evaluates the user's answer.
	CheckAnswer(ctx context.Context, req domain.CheckRequest) (domain.CheckResult, error)
}

// UserSettingsView is the narrow projection of user settings the tutor
// module consumes. Owned by auth/domain; an adapter in main.go projects
// the full UserSettings down to this view at the DI seam.
type UserSettingsView struct {
	TargetLanguage   string
	ProficiencyLevel string
	VocabularyType   string
	AIModel          string
}

// UserView is the narrow projection of a user the tutor module consumes.
type UserView struct {
	DisplayName string
}

type SettingsReader interface {
	GetByUserID(ctx context.Context, userID string) (*UserSettingsView, error)
}

type UserReader interface {
	GetByID(ctx context.Context, id string) (*UserView, error)
}
