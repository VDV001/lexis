package main

import (
	"context"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	tutorUsecase "github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	vocabUsecase "github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
)

// authSettingsRepo / authUserRepo are the cross-module surface area: anything
// that returns auth-domain types and is shared with another module funnels
// through these adapters. main.go is the only place allowed to know both
// auth/domain and the consuming module's port.
type authSettingsRepo interface {
	GetByUserID(ctx context.Context, userID string) (*authDomain.UserSettings, error)
}

type authUserRepo interface {
	GetByID(ctx context.Context, id string) (*authDomain.User, error)
}

// vocabSettingsAdapter projects auth UserSettings down to the narrow view
// the vocabulary module's usecase package consumes.
type vocabSettingsAdapter struct{ inner authSettingsRepo }

func (a vocabSettingsAdapter) GetByUserID(ctx context.Context, userID string) (*vocabUsecase.UserSettingsView, error) {
	s, err := a.inner.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &vocabUsecase.UserSettingsView{TargetLanguage: s.TargetLanguage}, nil
}

// tutorSettingsAdapter projects auth UserSettings to the tutor module's view.
type tutorSettingsAdapter struct{ inner authSettingsRepo }

func (a tutorSettingsAdapter) GetByUserID(ctx context.Context, userID string) (*tutorUsecase.UserSettingsView, error) {
	s, err := a.inner.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &tutorUsecase.UserSettingsView{
		TargetLanguage:   s.TargetLanguage,
		ProficiencyLevel: s.ProficiencyLevel,
		VocabularyType:   s.VocabularyType,
		AIModel:          s.AIModel,
	}, nil
}

// tutorUserAdapter projects auth User to the tutor module's view.
type tutorUserAdapter struct{ inner authUserRepo }

func (a tutorUserAdapter) GetByID(ctx context.Context, id string) (*tutorUsecase.UserView, error) {
	u, err := a.inner.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &tutorUsecase.UserView{DisplayName: u.DisplayName}, nil
}
