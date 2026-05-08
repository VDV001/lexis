package infra

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

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

type replyStreamState struct {
	lastReply string
}

// streamReplyDelta extracts the current reply preview from the accumulated JSON
// and emits only the newly appended reply text.
func streamReplyDelta(ctx context.Context, ch chan<- domain.ChatDelta, text string, state *replyStreamState) bool {
	reply, _, ok := extractReplyPreview(text)
	if !ok {
		return true
	}

	// If the preview changed unexpectedly, resync without emitting malformed text.
	if !strings.HasPrefix(reply, state.lastReply) {
		state.lastReply = reply
		return true
	}

	delta := reply[len(state.lastReply):]
	state.lastReply = reply
	if delta == "" {
		return true
	}

	return sendDelta(ctx, ch, domain.ChatDelta{Type: "delta", Content: delta})
}

// extractReplyPreview returns the currently decodable reply text from a partial
// structured JSON response. ok=false means the reply field was not found.
func extractReplyPreview(text string) (reply string, done bool, ok bool) {
	keyIdx := strings.Index(text, `"reply"`)
	if keyIdx == -1 {
		return "", false, false
	}

	i := keyIdx + len(`"reply"`)
	for i < len(text) && isJSONWhitespace(text[i]) {
		i++
	}
	if i >= len(text) || text[i] != ':' {
		return "", false, false
	}

	i++
	for i < len(text) && isJSONWhitespace(text[i]) {
		i++
	}
	if i >= len(text) || text[i] != '"' {
		return "", false, false
	}

	i++
	var b strings.Builder

	for i < len(text) {
		switch text[i] {
		case '"':
			return b.String(), true, true
		case '\\':
			if i+1 >= len(text) {
				return b.String(), false, true
			}

			decoded, width, complete := decodeJSONEscape(text[i:])
			if !complete {
				return b.String(), false, true
			}
			b.WriteString(decoded)
			i += width
			continue
		default:
			r, size := utf8.DecodeRuneInString(text[i:])
			if r == utf8.RuneError && size == 1 {
				return b.String(), false, true
			}
			b.WriteRune(r)
			i += size
		}
	}

	return b.String(), false, true
}

func isJSONWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t'
}

func decodeJSONEscape(s string) (decoded string, width int, complete bool) {
	if len(s) < 2 || s[0] != '\\' {
		return "", 0, false
	}

	switch s[1] {
	case '"', '\\', '/':
		return string(s[1]), 2, true
	case 'b':
		return "\b", 2, true
	case 'f':
		return "\f", 2, true
	case 'n':
		return "\n", 2, true
	case 'r':
		return "\r", 2, true
	case 't':
		return "\t", 2, true
	case 'u':
		return decodeUnicodeEscape(s)
	default:
		return "", 0, false
	}
}

func decodeUnicodeEscape(s string) (decoded string, width int, complete bool) {
	if len(s) < 6 {
		return "", 0, false
	}

	value, err := strconv.ParseUint(s[2:6], 16, 64)
	if err != nil {
		return "", 0, false
	}

	r := rune(value)
	width = 6

	if utf16.IsSurrogate(r) {
		if len(s) < 12 || s[6] != '\\' || s[7] != 'u' {
			return "", 0, false
		}

		nextValue, err := strconv.ParseUint(s[8:12], 16, 64)
		if err != nil {
			return "", 0, false
		}

		r = utf16.DecodeRune(r, rune(nextValue))
		if r == utf8.RuneError {
			return "", 0, false
		}
		width = 12
	}

	return string(r), width, true
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
