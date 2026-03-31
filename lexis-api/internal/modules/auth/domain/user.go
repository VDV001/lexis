package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	AvatarURL    *string
	CreatedAt    time.Time
	DeletedAt    *time.Time
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
