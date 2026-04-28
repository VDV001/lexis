package infra

import (
	"fmt"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
)

// ProviderRegistry maps model IDs to their AIProvider implementations.
type ProviderRegistry struct {
	providers map[string]usecase.AIProvider
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{providers: make(map[string]usecase.AIProvider)}
}

func (r *ProviderRegistry) Register(modelID string, provider usecase.AIProvider) {
	r.providers[modelID] = provider
}

func (r *ProviderRegistry) Get(modelID string) (usecase.AIProvider, error) {
	p, ok := r.providers[modelID]
	if !ok {
		return nil, fmt.Errorf("unknown model: %s", modelID)
	}
	return p, nil
}

// NewDefaultRegistry creates a ProviderRegistry pre-populated with all
// supported models. Pass empty strings for API keys of providers you
// don't want to register.
func NewDefaultRegistry(anthropicKey, openaiKey, qwenKey, geminiKey string) *ProviderRegistry {
	r := NewProviderRegistry()

	if anthropicKey != "" {
		claude := NewClaudeProvider(anthropicKey)
		r.Register("claude-sonnet-4-20250514", claude)
		r.Register("claude-haiku-4-20250514", claude)
	}
	if openaiKey != "" {
		openai := NewOpenAICompatibleProvider(openaiKey, "https://api.openai.com/v1/chat/completions")
		r.Register("gpt-4o", openai)
		r.Register("gpt-4o-mini", openai)
	}
	if qwenKey != "" {
		qwen := NewOpenAICompatibleProvider(qwenKey, qwenBaseURL)
		r.Register("qwen-plus", qwen)
	}
	if geminiKey != "" {
		gemini := NewGeminiProvider(geminiKey)
		r.Register("gemini-2.0-flash", gemini)
	}

	return r
}
