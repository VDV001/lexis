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

func TestNewRefreshToken(t *testing.T) {
	expiresAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		userID    string
		tokenHash string
		wantErr   error
	}{
		{name: "valid", userID: "user-123", tokenHash: "abc123hash", wantErr: nil},
		{name: "empty userID", userID: "", tokenHash: "abc123hash", wantErr: ErrUserIDRequired},
		{name: "empty tokenHash", userID: "user-123", tokenHash: "", wantErr: ErrTokenHashRequired},
		{name: "both empty", userID: "", tokenHash: "", wantErr: ErrUserIDRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := NewRefreshToken(tt.userID, tt.tokenHash, expiresAt, "Mozilla/5.0", "192.168.1.1")
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, token)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.userID, token.UserID)
			assert.Equal(t, tt.tokenHash, token.TokenHash)
			assert.Equal(t, expiresAt, token.ExpiresAt)
			assert.Equal(t, "Mozilla/5.0", token.UserAgent)
			assert.Equal(t, "192.168.1.1", token.IPAddress)
			assert.Nil(t, token.RevokedAt)
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
