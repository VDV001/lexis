package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRefreshToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		now       time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			now:       time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
			now:       time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name:      "exactly at expiry",
			expiresAt: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC),
			now:       time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &RefreshToken{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.want, token.IsExpired(tt.now))
		})
	}
}

func TestRefreshToken_IsRevoked(t *testing.T) {
	t.Run("not revoked", func(t *testing.T) {
		token := &RefreshToken{}
		assert.False(t, token.IsRevoked())
	})

	t.Run("revoked", func(t *testing.T) {
		now := time.Now()
		token := &RefreshToken{RevokedAt: &now}
		assert.True(t, token.IsRevoked())
	})
}
