package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

type stubUserRepo struct {
	user      *domain.User
	getErr    error
	createErr error
	updateErr error
}

func (s *stubUserRepo) Create(_ context.Context, u *domain.User) error {
	s.user = u
	return s.createErr
}
func (s *stubUserRepo) GetByID(_ context.Context, _ string) (*domain.User, error) {
	return s.user, s.getErr
}
func (s *stubUserRepo) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	return s.user, s.getErr
}
func (s *stubUserRepo) Update(_ context.Context, u *domain.User) error {
	s.user = u
	return s.updateErr
}

type stubSettingsRepo struct {
	settings  *domain.UserSettings
	getErr    error
	upsertErr error
}

func (s *stubSettingsRepo) GetByUserID(_ context.Context, _ string) (*domain.UserSettings, error) {
	return s.settings, s.getErr
}
func (s *stubSettingsRepo) Upsert(_ context.Context, settings *domain.UserSettings) error {
	s.settings = settings
	return s.upsertErr
}

func TestUserService_GetProfile(t *testing.T) {
	user := &domain.User{ID: "u1", Email: "test@test.com", DisplayName: "Test"}

	t.Run("success", func(t *testing.T) {
		svc := NewUserService(&stubUserRepo{user: user}, &stubSettingsRepo{})
		result, err := svc.GetProfile(context.Background(), "u1")
		require.NoError(t, err)
		assert.Equal(t, "u1", result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		svc := NewUserService(&stubUserRepo{getErr: domain.ErrUserNotFound}, &stubSettingsRepo{})
		_, err := svc.GetProfile(context.Background(), "u1")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestUserService_UpdateProfile(t *testing.T) {
	name := "New Name"
	avatar := "https://example.com/img.png"

	t.Run("update display name", func(t *testing.T) {
		repo := &stubUserRepo{user: &domain.User{ID: "u1", DisplayName: "Old"}}
		svc := NewUserService(repo, &stubSettingsRepo{})
		result, err := svc.UpdateProfile(context.Background(), "u1", UpdateProfileInput{DisplayName: &name})
		require.NoError(t, err)
		assert.Equal(t, "New Name", result.DisplayName)
	})

	t.Run("update avatar", func(t *testing.T) {
		repo := &stubUserRepo{user: &domain.User{ID: "u1"}}
		svc := NewUserService(repo, &stubSettingsRepo{})
		result, err := svc.UpdateProfile(context.Background(), "u1", UpdateProfileInput{AvatarURL: &avatar})
		require.NoError(t, err)
		assert.Equal(t, &avatar, result.AvatarURL)
	})

	t.Run("invalid display name", func(t *testing.T) {
		short := "a"
		repo := &stubUserRepo{user: &domain.User{ID: "u1"}}
		svc := NewUserService(repo, &stubSettingsRepo{})
		_, err := svc.UpdateProfile(context.Background(), "u1", UpdateProfileInput{DisplayName: &short})
		assert.ErrorIs(t, err, domain.ErrInvalidDisplayName)
	})

	t.Run("avatar too long", func(t *testing.T) {
		long := string(make([]byte, 2049))
		repo := &stubUserRepo{user: &domain.User{ID: "u1"}}
		svc := NewUserService(repo, &stubSettingsRepo{})
		_, err := svc.UpdateProfile(context.Background(), "u1", UpdateProfileInput{AvatarURL: &long})
		assert.ErrorIs(t, err, domain.ErrAvatarURLTooLong)
	})

	t.Run("user not found", func(t *testing.T) {
		svc := NewUserService(&stubUserRepo{getErr: domain.ErrUserNotFound}, &stubSettingsRepo{})
		_, err := svc.UpdateProfile(context.Background(), "u1", UpdateProfileInput{DisplayName: &name})
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("update error", func(t *testing.T) {
		repo := &stubUserRepo{user: &domain.User{ID: "u1"}, updateErr: errors.New("db")}
		svc := NewUserService(repo, &stubSettingsRepo{})
		_, err := svc.UpdateProfile(context.Background(), "u1", UpdateProfileInput{DisplayName: &name})
		assert.Error(t, err)
	})
}

func TestUserService_GetSettings(t *testing.T) {
	defaults := domain.DefaultSettings("u1")

	t.Run("success", func(t *testing.T) {
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{settings: &defaults})
		result, err := svc.GetSettings(context.Background(), "u1")
		require.NoError(t, err)
		assert.Equal(t, "en", result.TargetLanguage)
	})

	t.Run("error", func(t *testing.T) {
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{getErr: errors.New("db")})
		_, err := svc.GetSettings(context.Background(), "u1")
		assert.Error(t, err)
	})
}

func TestUserService_UpdateSettings(t *testing.T) {
	t.Run("valid settings", func(t *testing.T) {
		settings := domain.DefaultSettings("u1")
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{})
		err := svc.UpdateSettings(context.Background(), "u1", &settings)
		assert.NoError(t, err)
	})

	t.Run("invalid settings", func(t *testing.T) {
		settings := domain.DefaultSettings("u1")
		settings.TargetLanguage = "xx"
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{})
		err := svc.UpdateSettings(context.Background(), "u1", &settings)
		assert.ErrorIs(t, err, domain.ErrInvalidSettings)
	})

	t.Run("upsert error", func(t *testing.T) {
		settings := domain.DefaultSettings("u1")
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{upsertErr: errors.New("db")})
		err := svc.UpdateSettings(context.Background(), "u1", &settings)
		assert.Error(t, err)
	})
}

func TestUserService_PatchSettings(t *testing.T) {
	t.Run("partial update merges with existing", func(t *testing.T) {
		defaults := domain.DefaultSettings("u1")
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{settings: &defaults})

		newLevel := "c1"
		newGoal := 5000
		result, err := svc.PatchSettings(context.Background(), "u1", PatchSettingsInput{
			ProficiencyLevel: &newLevel,
			VocabGoal:        &newGoal,
		})
		require.NoError(t, err)
		assert.Equal(t, "c1", result.ProficiencyLevel)
		assert.Equal(t, 5000, result.VocabGoal)
		// Unchanged fields retain defaults.
		assert.Equal(t, "en", result.TargetLanguage)
		assert.Equal(t, "tech", result.VocabularyType)
		assert.Equal(t, "claude-sonnet-4-20250514", result.AIModel)
		assert.Equal(t, "ru", result.UILanguage)
	})

	t.Run("all fields", func(t *testing.T) {
		defaults := domain.DefaultSettings("u1")
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{settings: &defaults})

		lang := "en"
		level := "a2"
		vocab := "literary"
		model := "gpt-4o"
		goal := 10000
		ui := "en"
		result, err := svc.PatchSettings(context.Background(), "u1", PatchSettingsInput{
			TargetLanguage:   &lang,
			ProficiencyLevel: &level,
			VocabularyType:   &vocab,
			AIModel:          &model,
			VocabGoal:        &goal,
			UILanguage:       &ui,
		})
		require.NoError(t, err)
		assert.Equal(t, "en", result.TargetLanguage)
		assert.Equal(t, "a2", result.ProficiencyLevel)
		assert.Equal(t, "literary", result.VocabularyType)
		assert.Equal(t, "gpt-4o", result.AIModel)
		assert.Equal(t, 10000, result.VocabGoal)
		assert.Equal(t, "en", result.UILanguage)
	})

	t.Run("invalid value", func(t *testing.T) {
		defaults := domain.DefaultSettings("u1")
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{settings: &defaults})

		bad := "z9"
		_, err := svc.PatchSettings(context.Background(), "u1", PatchSettingsInput{
			ProficiencyLevel: &bad,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidSettings)
	})

	t.Run("get error", func(t *testing.T) {
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{getErr: errors.New("db")})
		_, err := svc.PatchSettings(context.Background(), "u1", PatchSettingsInput{})
		assert.Error(t, err)
	})

	t.Run("upsert error", func(t *testing.T) {
		defaults := domain.DefaultSettings("u1")
		svc := NewUserService(&stubUserRepo{}, &stubSettingsRepo{settings: &defaults, upsertErr: errors.New("db")})
		_, err := svc.PatchSettings(context.Background(), "u1", PatchSettingsInput{})
		assert.Error(t, err)
	})
}
