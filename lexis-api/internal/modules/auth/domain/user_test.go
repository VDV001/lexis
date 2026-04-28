package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"a@b.c", false},
		{"", true},
		{"ab", true},
		{"no-at-sign", true},
		{"no@dot", true},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidEmail)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		wantErr bool
	}{
		{"valid", "12345678", false},
		{"too short", "1234567", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.pass)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidPassword)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		display string
		wantErr bool
	}{
		{"valid", "Test User", false},
		{"min length", "ab", false},
		{"too short", "a", true},
		{"empty", "", true},
		{"max length", string(make([]byte, 100)), false},
		{"too long", string(make([]byte, 101)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDisplayName(tt.display)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDisplayName)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserSettings_Validate(t *testing.T) {
	valid := func() UserSettings {
		return DefaultSettings("user-1")
	}

	t.Run("valid defaults", func(t *testing.T) {
		s := valid()
		assert.NoError(t, s.Validate())
	})

	t.Run("invalid language", func(t *testing.T) {
		s := valid()
		s.TargetLanguage = "xx"
		assert.ErrorIs(t, s.Validate(), ErrInvalidSettings)
	})

	t.Run("invalid level", func(t *testing.T) {
		s := valid()
		s.ProficiencyLevel = "d1"
		assert.ErrorIs(t, s.Validate(), ErrInvalidSettings)
	})

	t.Run("invalid vocab type", func(t *testing.T) {
		s := valid()
		s.VocabularyType = "unknown"
		assert.ErrorIs(t, s.Validate(), ErrInvalidSettings)
	})

	t.Run("invalid model", func(t *testing.T) {
		s := valid()
		s.AIModel = "gpt-3"
		assert.ErrorIs(t, s.Validate(), ErrInvalidSettings)
	})

	t.Run("vocab goal too low", func(t *testing.T) {
		s := valid()
		s.VocabGoal = 50
		assert.ErrorIs(t, s.Validate(), ErrInvalidSettings)
	})

	t.Run("vocab goal too high", func(t *testing.T) {
		s := valid()
		s.VocabGoal = 100000
		assert.ErrorIs(t, s.Validate(), ErrInvalidSettings)
	})

	t.Run("invalid UI language", func(t *testing.T) {
		s := valid()
		s.UILanguage = "de"
		assert.ErrorIs(t, s.Validate(), ErrInvalidSettings)
	})
}

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
