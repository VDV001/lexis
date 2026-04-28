package usecase

import (
	"context"
	"fmt"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
)

type ChatService struct {
	registry domain.ProviderRegistry
	settings SettingsReader
	users    UserReader
}

func NewChatService(registry domain.ProviderRegistry, settings SettingsReader, users UserReader) *ChatService {
	return &ChatService{registry: registry, settings: settings, users: users}
}

type ChatInput struct {
	UserID   string
	Messages []domain.Message
}

func (s *ChatService) Chat(ctx context.Context, input ChatInput) (<-chan domain.ChatDelta, error) {
	settings, err := s.settings.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	user, err := s.users.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	provider, err := s.registry.Get(settings.AIModel)
	if err != nil {
		return nil, fmt.Errorf("get provider: %w", err)
	}

	promptSettings := domain.PromptSettings{
		UserName:         user.DisplayName,
		TargetLanguage:   settings.TargetLanguage,
		ProficiencyLevel: settings.ProficiencyLevel,
		VocabularyType:   settings.VocabularyType,
	}

	systemPrompt := domain.BuildSystemPrompt(promptSettings, domain.ModeChat)

	req := domain.ChatRequest{
		UserID:    input.UserID,
		Messages:  input.Messages,
		System:    systemPrompt,
		Model:     settings.AIModel,
		MaxTokens: 1024,
	}

	return provider.Chat(ctx, req)
}
