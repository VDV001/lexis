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

func TestChatService_Chat(t *testing.T) {
	registry := infra.NewProviderRegistry()
	registry.Register("test-model", &mockProvider{})

	svc := usecase.NewChatService(registry, &mockSettingsRepo{}, &mockUserRepo{})

	ch, err := svc.Chat(context.Background(), usecase.ChatInput{
		UserID: "user-123",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
	})
	require.NoError(t, err)

	var deltas []domain.ChatDelta
	for d := range ch {
		deltas = append(deltas, d)
	}

	assert.GreaterOrEqual(t, len(deltas), 2)
	assert.Equal(t, "delta", deltas[0].Type)
	assert.Equal(t, "Hello! ", deltas[0].Content)
}
