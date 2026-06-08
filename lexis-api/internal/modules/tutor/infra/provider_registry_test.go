package infra

import "testing"

func TestNewDefaultRegistry_Empty(t *testing.T) {
	r := NewDefaultRegistry("", "", "", "", "")
	if !r.Empty() {
		t.Fatal("registry with no keys must report Empty() == true")
	}
}

func TestNewDefaultRegistry_NativeProviderRegistered(t *testing.T) {
	r := NewDefaultRegistry("anthropic-key", "", "", "", "")
	if r.Empty() {
		t.Fatal("registry with a native key must not be Empty()")
	}
	if _, err := r.Get("claude-sonnet-4-20250514"); err != nil {
		t.Fatalf("native model should resolve: %v", err)
	}
}

func TestRegistry_OpenRouterRoutesExternalSlugs(t *testing.T) {
	r := NewDefaultRegistry("", "", "", "", "openrouter-key")

	if r.Empty() {
		t.Fatal("registry with an OpenRouter key must not be Empty()")
	}

	got1, err := r.Get("openai/gpt-4o-mini")
	if err != nil {
		t.Fatalf("external slug should route to OpenRouter: %v", err)
	}
	got2, err := r.Get("anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("second external slug should also route to OpenRouter: %v", err)
	}
	if got1 != got2 {
		t.Fatal("all external slugs must route to the single shared OpenRouter provider")
	}
}

func TestRegistry_NoOpenRouter_ExternalSlugUnknown(t *testing.T) {
	r := NewDefaultRegistry("anthropic-key", "", "", "", "")
	if _, err := r.Get("openai/gpt-4o-mini"); err == nil {
		t.Fatal("external slug must be unknown when OpenRouter is not configured")
	}
}

func TestRegistry_UnknownBareModel(t *testing.T) {
	r := NewDefaultRegistry("", "", "", "", "openrouter-key")
	if _, err := r.Get("totally-unknown"); err == nil {
		t.Fatal("a bare model with no slash must not route to OpenRouter")
	}
}
