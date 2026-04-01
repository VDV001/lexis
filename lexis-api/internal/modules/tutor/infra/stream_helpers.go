package infra

import (
	"context"

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
