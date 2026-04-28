package usecase

import (
	"context"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

// UserService handles user profile and settings operations.
type UserService struct {
	users    UserRepository
	settings SettingsRepository
}

func NewUserService(users UserRepository, settings SettingsRepository) *UserService {
	return &UserService{users: users, settings: settings}
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	return s.users.GetByID(ctx, userID)
}

type UpdateProfileInput struct {
	DisplayName *string
	AvatarURL   *string
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.DisplayName != nil {
		if err := domain.ValidateDisplayName(*input.DisplayName); err != nil {
			return nil, err // wraps ErrInvalidDisplayName
		}
		user.DisplayName = *input.DisplayName
	}
	if input.AvatarURL != nil {
		if len(*input.AvatarURL) > 2048 {
			return nil, domain.ErrAvatarURLTooLong
		}
		user.AvatarURL = input.AvatarURL
	}


	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetSettings(ctx context.Context, userID string) (*domain.UserSettings, error) {
	return s.settings.GetByUserID(ctx, userID)
}

func (s *UserService) UpdateSettings(ctx context.Context, userID string, settings *domain.UserSettings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	return s.settings.Upsert(ctx, settings)
}

// PatchSettingsInput carries optional fields for partial settings update.
// nil pointers mean "do not change".
type PatchSettingsInput struct {
	TargetLanguage   *string
	ProficiencyLevel *string
	VocabularyType   *string
	AIModel          *string
	VocabGoal        *int
	UILanguage       *string
}

// PatchSettings reads current settings, merges the patch, validates, and persists.
func (s *UserService) PatchSettings(ctx context.Context, userID string, input PatchSettingsInput) (*domain.UserSettings, error) {
	existing, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.TargetLanguage != nil {
		existing.TargetLanguage = *input.TargetLanguage
	}
	if input.ProficiencyLevel != nil {
		existing.ProficiencyLevel = *input.ProficiencyLevel
	}
	if input.VocabularyType != nil {
		existing.VocabularyType = *input.VocabularyType
	}
	if input.AIModel != nil {
		existing.AIModel = *input.AIModel
	}
	if input.VocabGoal != nil {
		existing.VocabGoal = *input.VocabGoal
	}
	if input.UILanguage != nil {
		existing.UILanguage = *input.UILanguage
	}

	if err := existing.Validate(); err != nil {
		return nil, err
	}
	if err := s.settings.Upsert(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}
