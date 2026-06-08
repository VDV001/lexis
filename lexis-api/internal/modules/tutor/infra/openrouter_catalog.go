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
	apiKey       string
	modelsURL    string
	ttl          time.Duration
	maxRetries   int
	retryBackoff time.Duration
	client       *http.Client

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
		apiKey:       apiKey,
		modelsURL:    modelsURL,
		ttl:          ttl,
		maxRetries:   2,
		retryBackoff: 200 * time.Millisecond,
		client:       &http.Client{Timeout: 15 * time.Second},
	}
}

// isTransientStatus reports whether an HTTP status warrants a retry: rate
// limiting (429) or server-side errors (5xx). Client errors (4xx, e.g. 401)
// are permanent and not retried.
func isTransientStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
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
	models, _ := v.([]usecase.RawCatalogModel)
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

// fetch performs the catalog GET with bounded retries on transient failures
// (network errors and 429/5xx). The GET is idempotent and free, so retrying is
// safe — unlike the chat/exercise POSTs, which are not retried to avoid
// double-charging a generation that may have succeeded server-side.
func (s *OpenRouterCatalogSource) fetch(ctx context.Context) ([]usecase.RawCatalogModel, error) {
	var lastErr error
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(s.retryBackoff * time.Duration(attempt)):
			}
		}
		models, status, err := s.fetchOnce(ctx)
		if err == nil {
			return models, nil
		}
		lastErr = err
		// A definite non-transient HTTP status is permanent — stop retrying.
		if status != 0 && !isTransientStatus(status) {
			return nil, err
		}
	}
	return nil, lastErr
}

// fetchOnce does a single catalog request. The returned status is the HTTP
// status code (0 for a transport-level error, which is treated as transient).
func (s *OpenRouterCatalogSource) fetchOnce(ctx context.Context) (models []usecase.RawCatalogModel, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.modelsURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create catalog request: %w", err)
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch OpenRouter catalog: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, resp.StatusCode, fmt.Errorf("OpenRouter catalog error %d: %s", resp.StatusCode, string(body))
	}

	var payload openRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode OpenRouter catalog: %w", err)
	}

	models = make([]usecase.RawCatalogModel, 0, len(payload.Data))
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
	return models, resp.StatusCode, nil
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
