package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		passHash    string
		displayName string
		wantErr     error
	}{
		{
			name:        "valid user",
			email:       "test@example.com",
			passHash:    "hashed",
			displayName: "Test User",
		},
		{
			name:        "invalid email",
			email:       "not-an-email",
			passHash:    "hashed",
			displayName: "Test",
			wantErr:     ErrInvalidEmail,
		},
		{
			name:        "empty display name",
			email:       "test@example.com",
			passHash:    "hashed",
			displayName: "",
			wantErr:     ErrDisplayNameRequired,
		},
		{
			name:        "display name too long",
			email:       "test@example.com",
			passHash:    "hashed",
			displayName: string(make([]byte, 101)),
			wantErr:     ErrDisplayNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.email, tt.passHash, tt.displayName)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.email, user.Email)
			assert.Equal(t, tt.passHash, user.PasswordHash)
			assert.Equal(t, tt.displayName, user.DisplayName)
		})
	}
}
