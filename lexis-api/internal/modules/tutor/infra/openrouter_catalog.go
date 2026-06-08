package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
)

const (
	openRouterModelsURL  = "https://openrouter.ai/api/v1/models"
	openRouterCatalogTTL = time.Hour
)

// OpenRouterCatalogSource fetches the OpenRouter model catalogue and caches it
// in memory for a TTL. A single self-hosted instance reads this only when the
// settings UI is opened, so an in-process cache (no Redis round-trip) is the
// right weight: it avoids hammering OpenRouter without adding infrastructure.
type OpenRouterCatalogSource struct {
	apiKey    string
	modelsURL string
	ttl       time.Duration
	client    *http.Client

	mu       sync.Mutex
	cached   []usecase.RawCatalogModel
	cachedAt time.Time
}

// NewOpenRouterCatalogSource builds a catalogue source pointed at the public
// OpenRouter models endpoint with the default cache TTL.
func NewOpenRouterCatalogSource(apiKey string) *OpenRouterCatalogSource {
	return newOpenRouterCatalogSource(apiKey, openRouterModelsURL, openRouterCatalogTTL)
}

func newOpenRouterCatalogSource(apiKey, modelsURL string, ttl time.Duration) *OpenRouterCatalogSource {
	return &OpenRouterCatalogSource{
		apiKey:    apiKey,
		modelsURL: modelsURL,
		ttl:       ttl,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

// List returns the catalogue, served from cache when a fetch is still fresh.
func (s *OpenRouterCatalogSource) List(ctx context.Context) ([]usecase.RawCatalogModel, error) {
	s.mu.Lock()
	if s.cached != nil && time.Since(s.cachedAt) < s.ttl {
		cached := s.cached
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	models, err := s.fetch(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cached = models
	s.cachedAt = time.Now()
	s.mu.Unlock()
	return models, nil
}

// openRouterModelsResponse is the relevant slice of the OpenRouter /models wire
// format. Unused fields (pricing, context_length, ...) are ignored.
type openRouterModelsResponse struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		Architecture struct {
			InputModalities  []string `json:"input_modalities"`
			OutputModalities []string `json:"output_modalities"`
			Modality         string   `json:"modality"`
		} `json:"architecture"`
	} `json:"data"`
}

func (s *OpenRouterCatalogSource) fetch(ctx context.Context) ([]usecase.RawCatalogModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create catalog request: %w", err)
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch OpenRouter catalog: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("OpenRouter catalog error %d: %s", resp.StatusCode, string(body))
	}

	var payload openRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode OpenRouter catalog: %w", err)
	}

	models := make([]usecase.RawCatalogModel, 0, len(payload.Data))
	for _, m := range payload.Data {
		in, out := m.Architecture.InputModalities, m.Architecture.OutputModalities
		if len(in) == 0 || len(out) == 0 {
			// Older entries only carry the combined "in->out" modality string.
			derivedIn, derivedOut := splitModality(m.Architecture.Modality)
			if len(in) == 0 {
				in = derivedIn
			}
			if len(out) == 0 {
				out = derivedOut
			}
		}
		models = append(models, usecase.RawCatalogModel{
			ID:               m.ID,
			Name:             m.Name,
			Description:      m.Description,
			InputModalities:  in,
			OutputModalities: out,
		})
	}
	return models, nil
}

// splitModality parses OpenRouter's combined modality string, e.g.
// "text+image->text", into input and output modality lists.
func splitModality(modality string) (in, out []string) {
	before, after, ok := strings.Cut(modality, "->")
	if !ok {
		return nil, nil
	}
	return strings.Split(before, "+"), strings.Split(after, "+")
}
