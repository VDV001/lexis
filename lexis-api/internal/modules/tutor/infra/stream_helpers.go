package infra

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
)

// sendDelta sends a delta to the channel, returning false if ctx is cancelled.
func sendDelta(ctx context.Context, ch chan<- domain.ChatDelta, delta domain.ChatDelta) bool {
	select {
	case ch <- delta:
		return true
	case <-ctx.Done():
		return false
	}
}

// parseStructuredResponse attempts to parse the full streamed text as JSON
// and emits structured deltas (correction, feedback, words).
func parseStructuredResponse(ctx context.Context, text string, ch chan<- domain.ChatDelta) {
	var resp struct {
		Reply      string             `json:"reply"`
		Correction *domain.Correction `json:"correction"`
		Feedback   *domain.Feedback   `json:"feedback"`
		ErrorType  *string            `json:"error_type"`
		NewWords   []string           `json:"new_words"`
	}

	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return
	}

	if resp.Correction != nil {
		if !sendDelta(ctx, ch, domain.ChatDelta{Type: "correction", Correction: resp.Correction}) {
			return
		}
	}
	if resp.Feedback != nil {
		if !sendDelta(ctx, ch, domain.ChatDelta{Type: "feedback", Feedback: resp.Feedback}) {
			return
		}
	}
	if len(resp.NewWords) > 0 {
		sendDelta(ctx, ch, domain.ChatDelta{Type: "words", Words: resp.NewWords})
	}
}
