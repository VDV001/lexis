package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

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

	sf       singleflight.Group
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
// On a cold or expired cache, concurrent callers collapse into a single
// upstream fetch (singleflight). If a refresh fails but a previous good cache
// exists, the stale cache is served (stale-while-error) so the UI keeps a live
// list; the error is logged for observability. Only a failure with no cache at
// all surfaces an error.
func (s *OpenRouterCatalogSource) List(ctx context.Context) ([]usecase.RawCatalogModel, error) {
	s.mu.Lock()
	if s.cached != nil && time.Since(s.cachedAt) < s.ttl {
		cached := s.cached
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	v, err, _ := s.sf.Do("models", func() (any, error) {
		models, fetchErr := s.fetch(ctx)
		if fetchErr != nil {
			return nil, fetchErr
		}
		s.mu.Lock()
		s.cached = models
		s.cachedAt = time.Now()
		s.mu.Unlock()
		return models, nil
	})
	if err != nil {
		s.mu.Lock()
		stale := s.cached
		s.mu.Unlock()
		if stale != nil {
			slog.Warn("openrouter catalog: serving stale cache after refresh failure", "error", err)
			return stale, nil
		}
		return nil, err
	}
	return v.([]usecase.RawCatalogModel), nil
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
