package usecase

import (
	"context"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
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

type SettingsReader interface {
	GetByUserID(ctx context.Context, userID string) (*authDomain.UserSettings, error)
}

type UserReader interface {
	GetByID(ctx context.Context, id string) (*authDomain.User, error)
}
