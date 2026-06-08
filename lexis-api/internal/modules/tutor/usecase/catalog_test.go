package usecase_test

import (
	"testing"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
)

func TestSelectUsableModels(t *testing.T) {
	raw := []usecase.RawCatalogModel{
		// Chat-capable, curated providers — kept.
		{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
		{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", InputModalities: []string{"text", "image"}, OutputModalities: []string{"text"}},
		{ID: "google/gemini-2.0-flash-001", Name: "Gemini 2.0 Flash", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
		{ID: "deepseek/deepseek-chat", Name: "DeepSeek Chat", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},

		// Not chat-capable — dropped.
		{ID: "openai/dall-e-3", Name: "DALL-E 3", InputModalities: []string{"text"}, OutputModalities: []string{"image"}},
		{ID: "openai/text-embedding-3-large", Name: "Embeddings", InputModalities: []string{"text"}, OutputModalities: []string{}},

		// Non-curated provider — dropped to keep the list short.
		{ID: "randomlab/exotic-model", Name: "Exotic", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},

		// Malformed id (no slash) — dropped.
		{ID: "bare-model", Name: "Bare", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
	}

	got := usecase.SelectUsableModels(raw)

	var ids []string
	for _, m := range got {
		ids = append(ids, m.ID)
	}

	// Expect exactly the four chat-capable curated models, ordered by provider
	// rank (openai, anthropic, google, deepseek).
	want := []string{
		"openai/gpt-4o-mini",
		"anthropic/claude-3.5-sonnet",
		"google/gemini-2.0-flash-001",
		"deepseek/deepseek-chat",
	}
	if len(ids) != len(want) {
		t.Fatalf("got %d models %v, want %d %v", len(ids), ids, len(want), want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %v, want %v", i, ids, want)
		}
	}

	// Provider must be extracted from the slug prefix.
	if got[0].Provider != "openai" {
		t.Errorf("provider extraction: got %q, want openai", got[0].Provider)
	}
}

func TestSelectUsableModels_SortsWithinProviderByID(t *testing.T) {
	raw := []usecase.RawCatalogModel{
		{ID: "openai/gpt-4o", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
		{ID: "openai/gpt-4o-mini", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
		{ID: "openai/gpt-3.5-turbo", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
	}
	got := usecase.SelectUsableModels(raw)
	want := []string{"openai/gpt-3.5-turbo", "openai/gpt-4o", "openai/gpt-4o-mini"}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("within-provider sort: got %v at %d, want %v", got[i].ID, i, want)
		}
	}
}
