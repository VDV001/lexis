package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/modules/progress/domain"
)

func TestNewSession(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		userID   string
		mode     domain.Mode
		language string
		level    string
		aiModel  string
		wantErr  bool
	}{
		{
			name:     "valid session",
			userID:   "user-1",
			mode:     domain.ModeChat,
			language: "en",
			level:    "B1",
			aiModel:  "claude-sonnet",
		},
		{
			name:     "invalid mode",
			userID:   "user-1",
			mode:     "invalid",
			language: "en",
			level:    "B1",
			aiModel:  "claude-sonnet",
			wantErr:  true,
		},
		{
			name:     "empty userID",
			userID:   "",
			mode:     domain.ModeQuiz,
			language: "en",
			level:    "B1",
			aiModel:  "claude-sonnet",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := domain.NewSession(tt.userID, tt.mode, tt.language, tt.level, tt.aiModel, now)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, s)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, s.ID)
			assert.Equal(t, tt.userID, s.UserID)
			assert.Equal(t, string(tt.mode), s.Mode)
			assert.Equal(t, now, s.StartedAt)
		})
	}
}

func TestNewRound(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	t.Run("valid round", func(t *testing.T) {
		r, err := domain.NewRound("session-1", "user-1", "chat", true, nil, "question", "answer", nil, nil, now)
		require.NoError(t, err)
		assert.NotEmpty(t, r.ID)
		assert.Equal(t, "session-1", r.SessionID)
		assert.Equal(t, now, r.CreatedAt)
	})

	t.Run("empty sessionID", func(t *testing.T) {
		r, err := domain.NewRound("", "user-1", "chat", true, nil, "q", "a", nil, nil, now)
		assert.Error(t, err)
		assert.Nil(t, r)
	})

	t.Run("empty userID", func(t *testing.T) {
		r, err := domain.NewRound("session-1", "", "chat", true, nil, "q", "a", nil, nil, now)
		assert.Error(t, err)
		assert.Nil(t, r)
	})
}
