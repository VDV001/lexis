package usecase

import (
	"context"
	"slices"
	"sort"
	"strings"
)

// CatalogModel is a single selectable AI model presented to the user.
type CatalogModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
}

// RawCatalogModel is the provider-neutral projection of an upstream catalogue
// entry (e.g. one OpenRouter model). The infra source maps the gateway's wire
// format down to this shape so the usecase filter stays gateway-agnostic.
type RawCatalogModel struct {
	ID               string
	Name             string
	Description      string
	InputModalities  []string
	OutputModalities []string
}

// CatalogSource fetches the raw upstream model catalogue.
type CatalogSource interface {
	List(ctx context.Context) ([]RawCatalogModel, error)
}

// ModelCatalogService turns a raw upstream catalogue into the curated list of
// selectable models, degrading gracefully: if the source fails or yields no
// usable model, it returns an embedded fallback shortlist so the settings UI is
// never empty. The upstream error is still returned so callers can log it.
type ModelCatalogService struct {
	source CatalogSource
}

func NewModelCatalogService(source CatalogSource) *ModelCatalogService {
	return &ModelCatalogService{source: source}
}

// List returns the curated selectable models. On upstream failure it returns
// (fallback, err); on success with an empty filtered result it returns
// (fallback, nil); otherwise (filtered, nil).
func (s *ModelCatalogService) List(ctx context.Context) ([]CatalogModel, error) {
	raw, err := s.source.List(ctx)
	if err != nil {
		return FallbackModels(), err
	}
	models := SelectUsableModels(raw)
	if len(models) == 0 {
		return FallbackModels(), nil
	}
	return models, nil
}

// FallbackModels is the small, hand-picked shortlist shown when the live
// OpenRouter catalogue is unavailable. These slugs are stable, widely used, and
// span a cheap default, a high-quality option, and a fast option.
func FallbackModels() []CatalogModel {
	return []CatalogModel{
		{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai", Description: "Fast and inexpensive default"},
		{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", Provider: "anthropic", Description: "High-quality reasoning"},
		{ID: "google/gemini-2.0-flash-001", Name: "Gemini 2.0 Flash", Provider: "google", Description: "Fast, capable"},
	}
}

// curatedProviders is the allowlist of provider prefixes shown to users. It
// keeps the selectable list short and recognisable instead of surfacing the
// full ~300-model OpenRouter catalogue.
var curatedProviders = map[string]int{
	"openai":     0,
	"anthropic":  1,
	"google":     2,
	"deepseek":   3,
	"qwen":       4,
	"meta-llama": 5,
	"mistralai":  6,
	"x-ai":       7,
	"cohere":     8,
}

// SelectUsableModels filters a raw catalogue to the chat-capable models of
// curated providers and returns them as CatalogModels sorted by provider rank
// then ID. "Usable" means the model accepts text input and produces text
// output, i.e. it can drive conversational tutoring and exercise generation;
// image-only, embedding, and other non-chat models are dropped.
func SelectUsableModels(raw []RawCatalogModel) []CatalogModel {
	out := make([]CatalogModel, 0, len(raw))
	for _, m := range raw {
		provider, _, ok := strings.Cut(m.ID, "/")
		if !ok {
			continue
		}
		if _, curated := curatedProviders[provider]; !curated {
			continue
		}
		if !containsText(m.InputModalities) || !containsText(m.OutputModalities) {
			continue
		}
		out = append(out, CatalogModel{
			ID:          m.ID,
			Name:        m.Name,
			Provider:    provider,
			Description: m.Description,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := curatedProviders[out[i].Provider], curatedProviders[out[j].Provider]
		if ri != rj {
			return ri < rj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func containsText(modalities []string) bool {
	return slices.Contains(modalities, "text")
}
