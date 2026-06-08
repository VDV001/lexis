package infra

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/stretchr/testify/assert"
)

// lineErrReader returns one line of data, then a non-EOF read error to simulate
// a connection dropping mid-stream.
type lineErrReader struct {
	line string
	sent bool
}

func (r *lineErrReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		n := copy(p, r.line)
		return n, nil
	}
	return 0, errors.New("connection reset by peer")
}

func collectDeltas(streamFn func(context.Context, io.Reader, chan<- domain.ChatDelta), body io.Reader) []domain.ChatDelta {
	ch := make(chan domain.ChatDelta, 64)
	go func() {
		streamFn(context.Background(), body, ch)
		close(ch)
	}()
	var out []domain.ChatDelta
	for d := range ch {
		out = append(out, d)
	}
	return out
}

func hasType(deltas []domain.ChatDelta, typ string) bool {
	for _, d := range deltas {
		if d.Type == typ {
			return true
		}
	}
	return false
}

func TestOpenAIStream_EmitsErrorOnMidStreamFailure(t *testing.T) {
	p := NewOpenAICompatibleProvider("k", "url")
	body := &lineErrReader{line: "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n"}

	deltas := collectDeltas(p.streamResponse, body)

	assert.True(t, hasType(deltas, "error"),
		"a mid-stream read failure must emit a terminal error delta; got %+v", deltas)
	assert.False(t, hasType(deltas, "done"),
		"a failed stream must not also report normal completion")
}

func TestOpenAIStream_CleanStreamEmitsDoneNotError(t *testing.T) {
	p := NewOpenAICompatibleProvider("k", "url")
	// A well-formed, fully-terminated stream.
	body := stringReader("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\ndata: [DONE]\n")

	deltas := collectDeltas(p.streamResponse, body)

	assert.True(t, hasType(deltas, "done"), "clean stream must report completion")
	assert.False(t, hasType(deltas, "error"), "clean stream must not emit an error delta")
}

func stringReader(s string) io.Reader { return &onceReader{s: s} }

type onceReader struct {
	s    string
	done bool
}

func (r *onceReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(p, r.s)
	return n, nil
}
