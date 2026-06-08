package infra

import (
	"fmt"
	"strings"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
)

// openRouterBaseURL is the OpenRouter OpenAI-compatible Chat Completions endpoint.
const openRouterBaseURL = "https://openrouter.ai/api/v1/chat/completions"

// ProviderRegistry maps model IDs to their AIProvider implementations.
//
// Native models (Claude, GPT-4o, Qwen, Gemini) are registered by exact ID.
// OpenRouter, when configured, is a single shared provider that serves the
// whole catalogue: any model ID in "provider/model" slug form that is not
// matched by an exact registration routes to it. Whether a given slug actually
// exists is resolved at the OpenRouter API boundary, not here.
type ProviderRegistry struct {
	providers  map[string]usecase.AIProvider
	openRouter usecase.AIProvider
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{providers: make(map[string]usecase.AIProvider)}
}

func (r *ProviderRegistry) Register(modelID string, provider usecase.AIProvider) {
	r.providers[modelID] = provider
}

// SetOpenRouter installs the shared OpenRouter provider used as the fallback
// for external model slugs.
func (r *ProviderRegistry) SetOpenRouter(provider usecase.AIProvider) {
	r.openRouter = provider
}

func (r *ProviderRegistry) Get(modelID string) (usecase.AIProvider, error) {
	if p, ok := r.providers[modelID]; ok {
		return p, nil
	}
	// External slugs ("provider/model") fall back to OpenRouter when configured.
	if r.openRouter != nil && strings.Contains(modelID, "/") {
		return r.openRouter, nil
	}
	return nil, fmt.Errorf("unknown model: %s", modelID)
}

// Empty reports whether the registry can serve no model at all: no native
// provider registered and no OpenRouter fallback. The composition root uses
// this to fail fast when the operator configured zero AI providers.
func (r *ProviderRegistry) Empty() bool {
	return len(r.providers) == 0 && r.openRouter == nil
}

// NewDefaultRegistry creates a ProviderRegistry pre-populated with all
// supported models. Pass empty strings for API keys of providers you
// don't want to register.
func NewDefaultRegistry(anthropicKey, openaiKey, qwenKey, geminiKey, openrouterKey string) *ProviderRegistry {
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
	if openrouterKey != "" {
		// HTTP-Referer and X-Title are OpenRouter's recommended attribution
		// headers; they identify this app on OpenRouter's rankings and are
		// harmless if absent. Native OpenAI/Qwen instances never send them.
		r.SetOpenRouter(NewOpenAICompatibleProviderWithHeaders(openrouterKey, openRouterBaseURL, map[string]string{
			"HTTP-Referer": "https://github.com/VDV001/lexis",
			"X-Title":      "Lexis",
		}))
	}

	return r
}
