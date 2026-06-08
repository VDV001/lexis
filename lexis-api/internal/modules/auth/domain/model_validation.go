package domain

import "strings"

// IsValidModel reports whether id is a model the system accepts for a user's
// settings. It accepts either a known native model (see ValidModels) or a
// well-formed external model slug in "provider/model" form, the shape used by
// OpenRouter and compatible gateways (e.g. "openai/gpt-4o-mini").
//
// The domain deliberately validates only the *shape* of an external slug.
// Whether a given slug actually exists in a provider's catalogue is a remote
// fact the domain cannot know; that existence check is enforced at the provider
// boundary, where an unknown model surfaces as a clear error on the AI call.
func IsValidModel(id string) bool {
	if ValidModels[id] {
		return true
	}
	return isExternalModelSlug(id)
}

// isExternalModelSlug reports whether id has the "provider/model" shape:
// exactly one slash with both segments non-empty and drawn from a conservative
// lowercase character set. The conservative set keeps bare identifiers such as
// "gpt-3" (no slash) invalid while admitting real catalogue slugs, including
// versioned and variant forms like "google/gemini-2.0-flash-001" and
// "meta-llama/llama-3.1-8b-instruct:free".
func isExternalModelSlug(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	provider, model, ok := strings.Cut(id, "/")
	if !ok || provider == "" || model == "" {
		return false
	}
	return isProviderSegment(provider) && isModelSegment(model)
}

func isProviderSegment(s string) bool {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

func isModelSegment(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-', r == '.', r == ':', r == '_':
		default:
			return false
		}
	}
	return true
}
