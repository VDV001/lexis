package infra

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
)

func TestParseStructuredResponse(t *testing.T) {
	ch := make(chan domain.ChatDelta, 4)

	parseStructuredResponse(context.Background(), `{"reply":"Hello there","correction":null,"feedback":{"type":"good","text":"Отлично"},"error_type":null,"new_words":["backend"]}`, ch)
	close(ch)

	var got []domain.ChatDelta
	for delta := range ch {
		got = append(got, delta)
	}

	require.Len(t, got, 2)
	assert.Equal(t, "feedback", got[0].Type)
	assert.Equal(t, "good", got[0].Feedback.Type)
	assert.Equal(t, "words", got[1].Type)
	assert.Equal(t, []string{"backend"}, got[1].Words)
}

func TestParseStructuredResponse_WithCodeFence(t *testing.T) {
	ch := make(chan domain.ChatDelta, 4)

	parseStructuredResponse(context.Background(), "```json\n{\"reply\":\"Hi\",\"correction\":null,\"feedback\":{\"type\":\"note\",\"text\":\"Норм\"},\"error_type\":null,\"new_words\":[]}\n```", ch)
	close(ch)

	var got []domain.ChatDelta
	for delta := range ch {
		got = append(got, delta)
	}

	require.Len(t, got, 1)
	assert.Equal(t, "feedback", got[0].Type)
	assert.Equal(t, "note", got[0].Feedback.Type)
}

func TestExtractReplyPreview_Partial(t *testing.T) {
	reply, done, ok := extractReplyPreview(`{"reply":"Hello, wor`)

	require.True(t, ok)
	assert.False(t, done)
	assert.Equal(t, "Hello, wor", reply)
}

func TestExtractReplyPreview_WithEscapes(t *testing.T) {
	reply, done, ok := extractReplyPreview(`{"reply":"Line 1\nLine 2 \"quoted\"","feedback":`)

	require.True(t, ok)
	assert.True(t, done)
	assert.Equal(t, "Line 1\nLine 2 \"quoted\"", reply)
}

func TestStreamReplyDelta_EmitsOnlyReplyGrowth(t *testing.T) {
	ch := make(chan domain.ChatDelta, 8)
	state := &replyStreamState{}

	require.True(t, streamReplyDelta(context.Background(), ch, `{"reply":"Hel`, state))
	require.True(t, streamReplyDelta(context.Background(), ch, `{"reply":"Hello"`, state))
	require.True(t, streamReplyDelta(context.Background(), ch, `{"reply":"Hello","feedback":null}`, state))
	close(ch)

	var got []domain.ChatDelta
	for delta := range ch {
		got = append(got, delta)
	}

	require.Len(t, got, 2)
	assert.Equal(t, domain.ChatDelta{Type: "delta", Content: "Hel"}, got[0])
	assert.Equal(t, domain.ChatDelta{Type: "delta", Content: "lo"}, got[1])
}
