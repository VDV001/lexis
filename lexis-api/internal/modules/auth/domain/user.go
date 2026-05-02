package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	AvatarURL    *string
	CreatedAt    time.Time
	DeletedAt    *time.Time
}

func NewUser(email, passwordHash, displayName string) (*User, error) {
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}
	if displayName == "" {
		return nil, ErrDisplayNameRequired
	}
	if len(displayName) > 100 {
		return nil, ErrDisplayNameTooLong
	}
	return &User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
	}, nil
}

type UserSettings struct {
	UserID           string
	TargetLanguage   string
	ProficiencyLevel string
	VocabularyType   string
	AIModel          string
	VocabGoal        int
	UILanguage       string
	UpdatedAt        time.Time
}

// Validation maps for settings domain invariants.
var (
	ValidLanguages  = map[string]bool{"en": true}
	ValidLevels     = map[string]bool{"a2": true, "b1": true, "b2": true, "c1": true}
	ValidVocabTypes = map[string]bool{"tech": true, "literary": true, "business": true}
	ValidModels     = map[string]bool{
		"claude-sonnet-4-20250514": true, "claude-haiku-4-20250514": true,
		"qwen-plus": true, "gpt-4o": true, "gpt-4o-mini": true, "gemini-2.0-flash": true,
	}
	ValidUILanguages = map[string]bool{"ru": true, "en": true}
)

// Validate checks all settings domain invariants.
func (s *UserSettings) Validate() error {
	if !ValidLanguages[s.TargetLanguage] {
		return fmt.Errorf("%w: invalid target_language: %q", ErrInvalidSettings, s.TargetLanguage)
	}
	if !ValidLevels[s.ProficiencyLevel] {
		return fmt.Errorf("%w: invalid proficiency_level: %q", ErrInvalidSettings, s.ProficiencyLevel)
	}
	if !ValidVocabTypes[s.VocabularyType] {
		return fmt.Errorf("%w: invalid vocabulary_type: %q", ErrInvalidSettings, s.VocabularyType)
	}
	if !ValidModels[s.AIModel] {
		return fmt.Errorf("%w: invalid ai_model: %q", ErrInvalidSettings, s.AIModel)
	}
	if s.VocabGoal < 100 || s.VocabGoal > 50000 {
		return fmt.Errorf("%w: vocab_goal must be between 100 and 50000, got %d", ErrInvalidSettings, s.VocabGoal)
	}
	if !ValidUILanguages[s.UILanguage] {
		return fmt.Errorf("%w: invalid ui_language: %q", ErrInvalidSettings, s.UILanguage)
	}
	return nil
}

// ValidateDisplayName checks display name constraints.
// Returns ErrInvalidDisplayName wrapping the specific reason.
func ValidateDisplayName(name string) error {
	if len(name) < 2 {
		return fmt.Errorf("%w: must be at least 2 characters", ErrInvalidDisplayName)
	}
	if len(name) > 100 {
		return fmt.Errorf("%w: must be at most 100 characters", ErrInvalidDisplayName)
	}
	return nil
}

// ValidateEmail checks email format and length.
func ValidateEmail(email string) error {
	if len(email) < 3 || len(email) > 255 {
		return fmt.Errorf("%w: must be between 3 and 255 characters", ErrInvalidEmail)
	}
	if !strings.Contains(email, "@") || !strings.Contains(email[strings.Index(email, "@")+1:], ".") {
		return fmt.Errorf("%w: invalid format", ErrInvalidEmail)
	}
	return nil
}

// ValidatePassword checks password length.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("%w: must be at least 8 characters", ErrInvalidPassword)
	}
	return nil
}

func DefaultSettings(userID string) UserSettings {
	return UserSettings{
		UserID:           userID,
		TargetLanguage:   "en",
		ProficiencyLevel: "b1",
		VocabularyType:   "tech",
		AIModel:          "claude-sonnet-4-20250514",
		VocabGoal:        3000,
		UILanguage:       "ru",
	}
}
