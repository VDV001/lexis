package domain

import "context"

// ProviderRegistry resolves a model ID to its AIProvider.
type ProviderRegistry interface {
	Get(modelID string) (AIProvider, error)
}

// AIProvider is the interface for all AI model providers
type AIProvider interface {
	// Chat streams a response for free practice mode
	Chat(ctx context.Context, req ChatRequest) (<-chan ChatDelta, error)

	// GenerateExercise generates a quiz/translate/gap/scramble exercise
	GenerateExercise(ctx context.Context, req ExerciseRequest) (Exercise, error)

	// CheckAnswer evaluates the user's answer
	CheckAnswer(ctx context.Context, req CheckRequest) (CheckResult, error)
}
