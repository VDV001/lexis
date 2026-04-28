package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVocabStatus_IsValid(t *testing.T) {
	tests := []struct {
		status VocabStatus
		want   bool
	}{
		{StatusUnknown, true},
		{StatusUncertain, true},
		{StatusConfident, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestErrInvalidStatus(t *testing.T) {
	assert.ErrorIs(t, ErrInvalidStatus, ErrInvalidStatus)
}

func TestNewWord(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		userID   string
		word     string
		language string
		context  string
		wantErr  bool
	}{
		{
			name:     "valid word",
			userID:   "user-1",
			word:     "goroutine",
			language: "en",
			context:  "Go concurrency",
		},
		{
			name:     "empty word",
			userID:   "user-1",
			word:     "",
			language: "en",
			wantErr:  true,
		},
		{
			name:     "empty userID",
			userID:   "",
			word:     "test",
			language: "en",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewWord(tt.userID, tt.word, tt.language, tt.context, now)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, w)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, w.ID)
			assert.Equal(t, tt.userID, w.UserID)
			assert.Equal(t, tt.word, w.Word)
			assert.Equal(t, tt.language, w.Language)
			assert.Equal(t, StatusUnknown, w.Status)
			assert.Equal(t, 2.5, w.EaseFactor)
			assert.Equal(t, now, w.NextReview)
			assert.Equal(t, now, w.LastSeen)
		})
	}
}
