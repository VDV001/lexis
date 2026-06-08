package domain

import "testing"

// TestIsValidModel pins the model-identifier invariant: a model is acceptable
// for a user's settings when it is either a known native model or a well-formed
// external "provider/model" slug (OpenRouter catalog shape). Existence of an
// external slug is a remote-catalog fact and is intentionally NOT validated
// here — only its shape.
func TestIsValidModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		// Native models stay valid (existing contract).
		{"native sonnet", "claude-sonnet-4-20250514", true},
		{"native gpt-4o", "gpt-4o", true},

		// Well-formed external slugs become valid.
		{"openrouter gpt-4o-mini", "openai/gpt-4o-mini", true},
		{"openrouter claude", "anthropic/claude-3.5-sonnet", true},
		{"openrouter gemini versioned", "google/gemini-2.0-flash-001", true},
		{"openrouter variant suffix", "meta-llama/llama-3.1-8b-instruct:free", true},
		{"openrouter deepseek", "deepseek/deepseek-chat", true},

		// Garbage without a slash stays invalid (protects existing tests:
		// "gpt-3", "nonexistent-model").
		{"bare gpt-3", "gpt-3", false},
		{"bare nonexistent", "nonexistent-model", false},
		{"empty", "", false},
		{"lone slash", "/", false},
		{"missing model", "openai/", false},
		{"missing provider", "/gpt-4o", false},
		{"uppercase rejected", "OpenAI/GPT-4o", false},
		{"double slash", "a//b", false},
		{"space in slug", "open ai/gpt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidModel(tt.model); got != tt.want {
				t.Errorf("IsValidModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
